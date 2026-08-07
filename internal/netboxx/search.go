package netboxx

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
)

// SearchGroup is one object type's slice of a global search result.
type SearchGroup struct {
	ObjectType string   `json:"object_type"`
	Count      int      `json:"count"`
	Results    []Object `json:"results"`
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

// Search runs a free-text query across every searchTarget concurrently and
// returns the non-empty groups. Results use NetBox's brief representation
// capped at perType per object type; Count carries the full match count so
// the caller knows when to narrow the query or use a dedicated list tool.
func (s *Service) Search(ctx context.Context, query string, perType int) ([]SearchGroup, error) {
	if perType < 1 {
		perType = 10
	}

	groups := make([]SearchGroup, len(searchTargets))
	errs := make([]error, len(searchTargets))
	var wg sync.WaitGroup
	for i, target := range searchTargets {
		wg.Go(func() {
			q := url.Values{}
			q.Set("q", query)
			q.Set("brief", "true")
			q.Set("limit", strconv.Itoa(perType))
			var env listEnvelope
			if err := s.get(ctx, target.path, q, &env); err != nil {
				errs[i] = fmt.Errorf("%s: %w", target.name, err)
				return
			}
			for _, obj := range env.Results {
				prune(obj)
			}
			groups[i] = SearchGroup{ObjectType: target.name, Count: env.Count, Results: env.Results}
		})
	}
	wg.Wait()

	if err := errors.Join(errs...); err != nil {
		return nil, err
	}

	out := make([]SearchGroup, 0, len(groups))
	for _, g := range groups {
		if g.Count > 0 {
			out = append(out, g)
		}
	}
	return out, nil
}
