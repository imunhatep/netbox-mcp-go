package netboxx

import (
	"context"
	"net/url"
)

// ListTenantGroups lists tenant groups. They nest, so parent narrows the
// listing to the direct children of one group (numeric id or slug).
func (s *Service) ListTenantGroups(ctx context.Context, search, parent string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "parent", parent)
	setBrief(q, brief)
	return list(ctx, s, "tenancy/tenant-groups", page, q)
}

// ListTenants lists tenants, optionally filtered by search text and group.
func (s *Service) ListTenants(ctx context.Context, search, group string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "group", group)
	setBrief(q, brief)
	return list(ctx, s, "tenancy/tenants", page, q)
}

// GetTenant fetches one tenant by numeric id or slug.
func (s *Service) GetTenant(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "tenancy/tenants", ref, "slug")
}

// ListContacts lists contacts — the people or teams attached to organizational
// objects. Group accepts a numeric id or slug.
func (s *Service) ListContacts(ctx context.Context, search, group string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "group", group)
	setBrief(q, brief)
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
