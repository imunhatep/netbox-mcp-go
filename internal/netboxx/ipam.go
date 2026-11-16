package netboxx

import (
	"context"
	"fmt"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// PrefixFilter narrows ListPrefixes.
type PrefixFilter struct {
	Search   string
	Within   string // parent prefix, e.g. 10.0.0.0/8 — matches contained subnets
	Contains string // IP or CIDR — matches the prefixes that contain it
	Site     string
	VRF      string // VRF id or route distinguisher (NetBox's vrf filter matches the RD, not the name)
	Status   string
	Family   string // "4" or "6"
	Tag      string
	Fields   []string // top-level fields to return; overrides Brief
	Brief    bool
}

// ListPrefixes lists IP prefixes (subnets) matching the filter.
func (s *Service) ListPrefixes(ctx context.Context, f PrefixFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setStr(q, "within_include", f.Within)
	setStr(q, "contains", f.Contains)
	setRef(q, "site", f.Site)
	setRef(q, "vrf", f.VRF)
	setStr(q, "status", f.Status)
	setStr(q, "family", f.Family)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	if err := setFields(q, f.Fields); err != nil {
		return nil, err
	}
	return list(ctx, s, "ipam/prefixes", page, q)
}

// GetPrefix fetches one prefix by numeric id or exact CIDR (e.g. 10.1.0.0/24).
func (s *Service) GetPrefix(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "ipam/prefixes", ref, "prefix")
}

// IPFilter narrows ListIPAddresses.
type IPFilter struct {
	Search  string
	Parent  string // containing prefix, e.g. 10.1.0.0/24
	Device  string // device the IP's interface belongs to (id or name)
	VM      string // virtual machine the IP's interface belongs to (id or name)
	VRF     string // VRF id or route distinguisher
	Status  string
	DNSName string
	Tag     string
	Fields  []string // top-level fields to return; overrides Brief
	Brief   bool
}

// ListIPAddresses lists IP addresses matching the filter.
func (s *Service) ListIPAddresses(ctx context.Context, f IPFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setStr(q, "parent", f.Parent)
	setRef(q, "device", f.Device)
	setRef(q, "virtual_machine", f.VM)
	setRef(q, "vrf", f.VRF)
	setStr(q, "status", f.Status)
	setStr(q, "dns_name", f.DNSName)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	if err := setFields(q, f.Fields); err != nil {
		return nil, err
	}
	return list(ctx, s, "ipam/ip-addresses", page, q)
}

// MaxLookupAddresses caps one LookupIPAddresses call.
const MaxLookupAddresses = 100

// lookupConcurrency bounds the NetBox requests one lookup has in flight.
const lookupConcurrency = 8

// prefixHintFields keeps the containing-prefix hint small: enough to say which
// network an unregistered address belongs to and who owns it.
const prefixHintFields = "id,prefix,status,vrf,tenant,role,scope_type,scope,vlan,description"

// IPLookup answers "what is this address": the IP objects registered for it
// and, when there are none, the prefixes that contain it — so a miss still
// says which network the address belongs to.
type IPLookup struct {
	Address            string   `json:"address"`
	Found              bool     `json:"found"`
	IPAddresses        []Object `json:"ip_addresses,omitempty"`
	ContainingPrefixes []Object `json:"containing_prefixes,omitempty"`
	Error              string   `json:"error,omitempty"`
}

// LookupIPAddresses resolves many addresses in one call. Each address is
// matched exactly (NetBox's address filter: a bare address matches any mask
// length, "a.b.c.d/nn" only that mask). For an address with no IP object the
// containing prefixes are returned instead, most general first. A failure on
// one address is reported in its Error and does not fail the others.
func (s *Service) LookupIPAddresses(ctx context.Context, addrs []string) ([]IPLookup, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%w: at least one address is required", ErrValidation)
	}
	if len(addrs) > MaxLookupAddresses {
		return nil, fmt.Errorf("%w: %d addresses, at most %d per call", ErrValidation, len(addrs), MaxLookupAddresses)
	}
	clean, hosts := make([]string, len(addrs)), make([]string, len(addrs))
	for i, a := range addrs {
		clean[i] = strings.TrimSpace(a)
		host, _, _ := strings.Cut(clean[i], "/")
		if _, err := netip.ParseAddr(host); err != nil {
			return nil, fmt.Errorf("%w: %q is not an IP address", ErrValidation, a)
		}
		hosts[i] = host
	}

	out := make([]IPLookup, len(clean))
	sem := make(chan struct{}, lookupConcurrency)
	var wg sync.WaitGroup
	for i := range clean {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = s.lookupIP(ctx, clean[i], hosts[i])
		})
	}
	wg.Wait()
	return out, ctx.Err()
}

func (s *Service) lookupIP(ctx context.Context, addr, host string) IPLookup {
	res := IPLookup{Address: addr}

	q := url.Values{}
	q.Set("address", addr)
	q.Set("limit", strconv.Itoa(s.pageSize))
	var ips listEnvelope
	if err := s.get(ctx, "ipam/ip-addresses", q, &ips); err != nil {
		res.Error = err.Error()
		return res
	}
	if len(ips.Results) > 0 {
		for _, obj := range ips.Results {
			prune(obj)
		}
		res.Found, res.IPAddresses = true, ips.Results
		return res
	}

	q = url.Values{}
	q.Set("contains", host)
	q.Set("fields", prefixHintFields)
	q.Set("limit", "10")
	var prefixes listEnvelope
	if err := s.get(ctx, "ipam/prefixes", q, &prefixes); err != nil {
		res.Error = "no IP object; containing prefixes: " + err.Error()
		return res
	}
	for _, obj := range prefixes.Results {
		prune(obj)
	}
	res.ContainingPrefixes = prefixes.Results
	return res
}

// GetIPAddress fetches one IP by numeric id or address. A bare address (no
// mask) matches any mask length; an exact "a.b.c.d/nn" also works.
func (s *Service) GetIPAddress(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "ipam/ip-addresses", ref, "address")
}

// VLANFilter narrows ListVLANs. Site accepts a numeric id or slug.
type VLANFilter struct {
	Search string
	Site   string
	VID    string // 802.1Q id, e.g. "100"
	Brief  bool
}

// ListVLANs lists VLANs matching the filter.
func (s *Service) ListVLANs(ctx context.Context, f VLANFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "site", f.Site)
	setStr(q, "vid", f.VID)
	setBrief(q, f.Brief)
	return list(ctx, s, "ipam/vlans", page, q)
}

// VRFFilter narrows ListVRFs.
type VRFFilter struct {
	Search string
	Brief  bool
}

// ListVRFs lists VRFs matching the filter.
func (s *Service) ListVRFs(ctx context.Context, f VRFFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setBrief(q, f.Brief)
	return list(ctx, s, "ipam/vrfs", page, q)
}
