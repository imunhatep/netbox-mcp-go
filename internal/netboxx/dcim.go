package netboxx

import (
	"context"
	"net/url"
)

// RegionFilter narrows ListRegions. Parent accepts a numeric id or slug.
type RegionFilter struct {
	Search string
	Parent string
	Brief  bool
}

// ListRegions lists regions. Regions nest, so Parent narrows the listing to the
// direct children of one region.
func (s *Service) ListRegions(ctx context.Context, f RegionFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "parent", f.Parent)
	setBrief(q, f.Brief)
	return list(ctx, s, "dcim/regions", page, q)
}

// GetRegion fetches one region by numeric id or slug.
func (s *Service) GetRegion(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/regions", ref, "slug")
}

// SiteGroupFilter narrows ListSiteGroups. Parent accepts a numeric id or slug.
type SiteGroupFilter struct {
	Search string
	Parent string
	Brief  bool
}

// ListSiteGroups lists site groups — the functional (non-geographic) grouping
// of sites. They nest, so Parent narrows to one group's direct children.
func (s *Service) ListSiteGroups(ctx context.Context, f SiteGroupFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "parent", f.Parent)
	setBrief(q, f.Brief)
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

// LocationFilter narrows ListLocations. Site/Parent accept a numeric id or slug.
type LocationFilter struct {
	Search string
	Site   string
	Parent string
	Brief  bool
}

// ListLocations lists locations (floors, rooms, cages) within sites. Locations
// nest, so Parent narrows to one location's direct children.
func (s *Service) ListLocations(ctx context.Context, f LocationFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "site", f.Site)
	setRef(q, "parent", f.Parent)
	setBrief(q, f.Brief)
	return list(ctx, s, "dcim/locations", page, q)
}

// GetLocation fetches one location by numeric id or slug.
func (s *Service) GetLocation(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/locations", ref, "slug")
}

// RackFilter narrows ListRacks. Site/Location accept a numeric id or slug.
type RackFilter struct {
	Search   string
	Site     string
	Location string
	Brief    bool
}

// ListRacks lists racks matching the filter.
func (s *Service) ListRacks(ctx context.Context, f RackFilter, page int) (*Page, error) {
	q := url.Values{}
	setStr(q, "q", f.Search)
	setRef(q, "site", f.Site)
	setRef(q, "location", f.Location)
	setBrief(q, f.Brief)
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
	IDs      []string // exact numeric ids, OR-ed: one call instead of a get per device
	Names    []string // exact names (case-insensitive), OR-ed
	Fields   []string // top-level fields to return; overrides Brief
	Brief    bool
}

// ListDevices lists devices matching the filter.
func (s *Service) ListDevices(ctx context.Context, f DeviceFilter, page int) (*Page, error) {
	q := url.Values{}
	setMulti(q, "id", f.IDs)
	setMulti(q, "name", f.Names)
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
	if err := setFields(q, f.Fields); err != nil {
		return nil, err
	}
	return list(ctx, s, "dcim/devices", page, q)
}

// GetDevice fetches one device by numeric id or exact name.
func (s *Service) GetDevice(ctx context.Context, ref string) (Object, error) {
	return getOne(ctx, s, "dcim/devices", ref, "name")
}

// DeviceInterfaceFilter narrows ListDeviceInterfaces. Device is required and
// accepts a numeric id or exact name.
type DeviceInterfaceFilter struct {
	Device string
	Brief  bool
}

// ListDeviceInterfaces lists the interfaces of one device.
func (s *Service) ListDeviceInterfaces(ctx context.Context, f DeviceInterfaceFilter, page int) (*Page, error) {
	q := url.Values{}
	setRef(q, "device", f.Device)
	setBrief(q, f.Brief)
	return list(ctx, s, "dcim/interfaces", page, q)
}
