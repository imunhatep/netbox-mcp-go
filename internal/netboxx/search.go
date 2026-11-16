package netboxx

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// SearchGroup is one object type's slice of a global search result. Error is
// set instead of Results when that one type failed; the other types are still
// returned.
type SearchGroup struct {
	ObjectType string   `json:"object_type"`
	Count      int      `json:"count"`
	Results    []Object `json:"results,omitempty"`
	Error      string   `json:"error,omitempty"`
}

// SearchFilter narrows Search.
type SearchFilter struct {
	Query   string
	Types   []string // searchTargets names to query; empty means chosen from the query's shape
	PerType int      // results per object type
}

// searchTargets are the endpoints a global search fans out over. NetBox's
// REST API has no cross-type search endpoint, so each type's own free-text
// `q` filter is queried concurrently.
var searchTargets = []struct{ name, path string }{
	{"regions", "dcim/regions"},
	{"site_groups", "dcim/site-groups"},
	{"sites", "dcim/sites"},
	{"locations", "dcim/locations"},
	{"racks", "dcim/racks"},
	{"devices", "dcim/devices"},
	{"prefixes", "ipam/prefixes"},
	{"ip_addresses", "ipam/ip-addresses"},
	{"vlans", "ipam/vlans"},
	{"vrfs", "ipam/vrfs"},
	{"clusters", "virtualization/clusters"},
	{"virtual_machines", "virtualization/virtual-machines"},
	{"tenant_groups", "tenancy/tenant-groups"},
	{"tenants", "tenancy/tenants"},
	{"contacts", "tenancy/contacts"},
}

// addressTypes are the only types whose free-text search matches an IP
// address or CIDR, so an address-shaped query skips the rest of the fan-out.
var addressTypes = []string{"prefixes", "ip_addresses"}

// SearchTypes lists the object type names Search accepts in SearchFilter.Types.
func SearchTypes() []string {
	out := make([]string, len(searchTargets))
	for i, t := range searchTargets {
		out[i] = t.name
	}
	return out
}

// searchTypesFor picks the types to query: the requested ones, or for an IP
// address or CIDR only addressTypes, or everything.
func searchTypesFor(f SearchFilter) ([]string, error) {
	if len(f.Types) > 0 {
		known := SearchTypes()
		for _, t := range f.Types {
			if !slices.Contains(known, t) {
				return nil, fmt.Errorf("%w: unknown search type %q, expected one of %s",
					ErrValidation, t, strings.Join(known, ", "))
			}
		}
		return f.Types, nil
	}
	q := strings.TrimSpace(f.Query)
	if _, err := netip.ParseAddr(q); err == nil {
		return addressTypes, nil
	}
	if _, err := netip.ParsePrefix(q); err == nil {
		return addressTypes, nil
	}
	return SearchTypes(), nil
}

// Search runs a free-text query across object types concurrently and returns
// the non-empty groups. Results use NetBox's brief representation capped at
// PerType per object type; Count carries the full match count so the caller
// knows when to narrow the query or use a dedicated list tool.
func (s *Service) Search(ctx context.Context, f SearchFilter) ([]SearchGroup, error) {
	perType := f.PerType
	if perType < 1 {
		perType = 10
	}
	types, err := searchTypesFor(f)
	if err != nil {
		return nil, err
	}

	var targets []struct{ name, path string }
	for _, t := range searchTargets {
		if slices.Contains(types, t.name) {
			targets = append(targets, t)
		}
	}

	groups := make([]SearchGroup, len(targets))
	errs := make([]error, len(targets))
	var wg sync.WaitGroup
	for i, target := range targets {
		wg.Go(func() {
			q := url.Values{}
			q.Set("q", f.Query)
			q.Set("brief", "true")
			q.Set("limit", strconv.Itoa(perType))
			var env listEnvelope
			if err := s.get(ctx, target.path, q, &env); err != nil {
				errs[i] = fmt.Errorf("%s: %w", target.name, err)
				groups[i] = SearchGroup{ObjectType: target.name, Error: err.Error()}
				return
			}
			for _, obj := range env.Results {
				prune(obj)
			}
			groups[i] = SearchGroup{ObjectType: target.name, Count: env.Count, Results: env.Results}
		})
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	failed := 0
	for _, e := range errs {
		if e != nil {
			failed++
		}
	}
	if failed == len(targets) {
		return nil, errors.Join(errs...)
	}

	out := make([]SearchGroup, 0, len(groups))
	for _, g := range groups {
		if g.Count > 0 || g.Error != "" {
			out = append(out, g)
		}
	}
	return out, nil
}
