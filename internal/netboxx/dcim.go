package netboxx

import (
	"context"
	"net/url"
)

// ListRegions lists regions. Regions nest, so parent narrows the listing to the
// direct children of one region (numeric id or slug).
func (s *Service) ListRegions(ctx context.Context, search, parent string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "parent", parent)
	setBrief(q, brief)
	return list(ctx, s, "dcim/regions", page, q)
}

// GetRegion fetches one region by numeric id or slug.
func (s *Service) GetRegion(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/regions", ref, "slug")
}

// ListSiteGroups lists site groups — the functional (non-geographic) grouping
// of sites. They nest, so parent narrows to one group's direct children.
func (s *Service) ListSiteGroups(ctx context.Context, search, parent string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "parent", parent)
	setBrief(q, brief)
	return list(ctx, s, "dcim/site-groups", page, q)
}

// GetSiteGroup fetches one site group by numeric id or slug.
func (s *Service) GetSiteGroup(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/site-groups", ref, "slug")
}

// SiteFilter narrows ListSites. Region/Group/Tenant accept a numeric id or slug.
type SiteFilter struct {
	Search string
	Region string
	Group  string
	Tenant string
	Status string
	Tag    string
	Brief  bool
}

// ListSites lists sites matching the filter.
func (s *Service) ListSites(ctx context.Context, f SiteFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "region", f.Region)
	setRef(q, "group", f.Group)
	setRef(q, "tenant", f.Tenant)
	setStr(q, "status", f.Status)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	return list(ctx, s, "dcim/sites", page, q)
}

// GetSite fetches one site by numeric id or slug.
func (s *Service) GetSite(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/sites", ref, "slug")
}

// ListLocations lists locations (floors, rooms, cages) within sites. Locations
// nest, so parent narrows to one location's direct children.
func (s *Service) ListLocations(ctx context.Context, search, site, parent string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "site", site)
	setRef(q, "parent", parent)
	setBrief(q, brief)
	return list(ctx, s, "dcim/locations", page, q)
}

// GetLocation fetches one location by numeric id or slug.
func (s *Service) GetLocation(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/locations", ref, "slug")
}

// ListRacks lists racks, optionally filtered by search text, site and location.
func (s *Service) ListRacks(ctx context.Context, search, site, location string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", search)
	setRef(q, "site", site)
	setRef(q, "location", location)
	setBrief(q, brief)
	return list(ctx, s, "dcim/racks", page, q)
}

// DeviceFilter narrows ListDevices. Reference fields accept a numeric id or
// slug/name.
type DeviceFilter struct {
	Search   string
	Region   string
	Site     string
	Location string
	Rack     string
	Role     string
	Tenant   string
	Status   string
	Tag      string
	Brief    bool
}

// ListDevices lists devices matching the filter.
func (s *Service) ListDevices(ctx context.Context, f DeviceFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "region", f.Region)
	setRef(q, "site", f.Site)
	setRef(q, "location", f.Location)
	setRef(q, "rack", f.Rack)
	setRef(q, "role", f.Role)
	setRef(q, "tenant", f.Tenant)
	setStr(q, "status", f.Status)
	setStr(q, "tag", f.Tag)
	setBrief(q, f.Brief)
	return list(ctx, s, "dcim/devices", page, q)
}

// GetDevice fetches one device by numeric id or exact name.
func (s *Service) GetDevice(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/devices", ref, "name")
}

// ListDeviceInterfaces lists the interfaces of a device (id or exact name).
func (s *Service) ListDeviceInterfaces(ctx context.Context, device string, brief bool, page int) (*Page, error) {
	q := url.Values{}
	setRef(q, "device", device)
	setBrief(q, brief)
	return list(ctx, s, "dcim/interfaces", page, q)
}
