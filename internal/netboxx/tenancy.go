package netboxx

import (
	"context"
	"net/url"
)

// TenantGroupFilter narrows ListTenantGroups. Parent accepts a numeric id or
// slug.
type TenantGroupFilter struct {
	Search string
	Parent string
	Brief  bool
}

// ListTenantGroups lists tenant groups. They nest, so Parent narrows the
// listing to the direct children of one group.
func (s *Service) ListTenantGroups(ctx context.Context, f TenantGroupFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "parent", f.Parent)
	setBrief(q, f.Brief)
	return list(ctx, s, "tenancy/tenant-groups", page, q)
}

// TenantFilter narrows ListTenants. Group accepts a numeric id or slug.
type TenantFilter struct {
	Search string
	Group  string
	Brief  bool
}

// ListTenants lists tenants matching the filter.
func (s *Service) ListTenants(ctx context.Context, f TenantFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "group", f.Group)
	setBrief(q, f.Brief)
	return list(ctx, s, "tenancy/tenants", page, q)
}

// GetTenant fetches one tenant by numeric id or slug.
func (s *Service) GetTenant(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "tenancy/tenants", ref, "slug")
}

// ContactFilter narrows ListContacts. Group accepts a numeric id or slug.
type ContactFilter struct {
	Search string
	Group  string
	Brief  bool
}

// ListContacts lists contacts — the people or teams attached to organizational
// objects.
func (s *Service) ListContacts(ctx context.Context, f ContactFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "group", f.Group)
	setBrief(q, f.Brief)
	return list(ctx, s, "tenancy/contacts", page, q)
}

// Status fetches NetBox version and plugin information from /api/status/.
func (s *Service) Status(ctx context.Context) (Object, error) {
	var st Object
	if err := s.get(ctx, "status", nil, &st); err != nil {
		return nil, err
	}
	prune(st)
	return st, nil
}
