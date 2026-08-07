package netboxx

import (
	"context"
	"net/url"
)

// PrefixFilter narrows ListPrefixes.
type PrefixFilter struct {
	Search string
	Within string // parent prefix, e.g. 10.0.0.0/8 — matches contained subnets
	Site   string
	VRF    string
	Status string
	Family string // "4" or "6"
	Tag    string
	Brief  bool
}

// ListPrefixes lists IP prefixes (subnets) matching the filter.
func (s *Service) ListPrefixes(ctx context.Context, f PrefixFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setStr(q, "within_include", f.Within)
	setRef(q, "site", f.Site)
	setRef(q, "vrf", f.VRF)
	setStr(q, "status", f.Status)
	setStr(q, "family", f.Family)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
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
	VRF     string
	Status  string
	DNSName string
	Tag     string
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
	return list(ctx, s, "ipam/ip-addresses", page, q)
}

// GetIPAddress fetches one IP by numeric id or address. A bare address (no
// mask) matches any mask length; an exact "a.b.c.d/nn" also works.
func (s *Service) GetIPAddress(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "ipam/ip-addresses", ref, "address")
}

// ListVLANs lists VLANs, optionally filtered by search text, site and 802.1Q id.
func (s *Service) ListVLANs(ctx context.Context, search, site, vid string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "site", site)
	setStr(q, "vid", vid)
	setBrief(q, brief)
	return list(ctx, s, "ipam/vlans", page, q)
}

// ListVRFs lists VRFs, optionally filtered by search text.
func (s *Service) ListVRFs(ctx context.Context, search string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setBrief(q, brief)
	return list(ctx, s, "ipam/vrfs", page, q)
}
