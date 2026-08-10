package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type DCIMBackend interface {
	ListRegions(ctx context.Context, f netboxx.RegionFilter, page int) (*netboxx.Page, error)
	GetRegion(ctx context.Context, ref string) (netboxx.Object, error)
	ListSiteGroups(ctx context.Context, f netboxx.SiteGroupFilter, page int) (*netboxx.Page, error)
	GetSiteGroup(ctx context.Context, ref string) (netboxx.Object, error)
	ListSites(ctx context.Context, f netboxx.SiteFilter, page int) (*netboxx.Page, error)
	GetSite(ctx context.Context, ref string) (netboxx.Object, error)
	ListLocations(ctx context.Context, f netboxx.LocationFilter, page int) (*netboxx.Page, error)
	GetLocation(ctx context.Context, ref string) (netboxx.Object, error)
	ListRacks(ctx context.Context, f netboxx.RackFilter, page int) (*netboxx.Page, error)
	ListDevices(ctx context.Context, f netboxx.DeviceFilter, page int) (*netboxx.Page, error)
	GetDevice(ctx context.Context, ref string) (netboxx.Object, error)
	ListDeviceInterfaces(ctx context.Context, f netboxx.DeviceInterfaceFilter, page int) (*netboxx.Page, error)
}

// briefDesc documents the shared optional "brief" argument on list tools.
const briefDesc = "return NetBox's brief representation (id, name and little else) instead of full objects"

// parentDesc documents the "parent" argument on the nesting organization types.
const parentDesc = "list only the direct children of this parent (id or slug); omit for the whole flat list"

// RegisterDCIM adds the organization hierarchy (regions, site groups, sites,
// locations) plus rack, device and interface tools and returns their names.
func RegisterDCIM(s *server.MCPServer, b DCIMBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_regions",
		mcp.WithDescription("List NetBox regions — the geographic hierarchy (e.g. continent > country > city) that sites belong to."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("parent", mcp.Description(parentDesc)),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.RegionFilter{
			Search: req.GetString("search", ""),
			Parent: req.GetString("parent", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListRegions(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_region",
		mcp.WithDescription("Get one region by numeric id or slug, with all fields including parent, site count, custom_fields and tags."),
		mcp.WithString("region", mcp.Required(), mcp.Description("region id or slug, e.g. 3 or western-europe")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("region")
		if err != nil {
			return errResult(err)
		}
		region, err := b.GetRegion(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(region)
	})

	add(mcp.NewTool("list_site_groups",
		mcp.WithDescription("List site groups — the functional grouping of sites (e.g. production, branch offices), independent of geography."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("parent", mcp.Description(parentDesc)),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.SiteGroupFilter{
			Search: req.GetString("search", ""),
			Parent: req.GetString("parent", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListSiteGroups(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_site_group",
		mcp.WithDescription("Get one site group by numeric id or slug, with all fields including parent and site count."),
		mcp.WithString("site_group", mcp.Required(), mcp.Description("site group id or slug, e.g. 2 or production")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("site_group")
		if err != nil {
			return errResult(err)
		}
		group, err := b.GetSiteGroup(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(group)
	})

	add(mcp.NewTool("list_sites",
		mcp.WithDescription("List NetBox sites (physical locations), filterable by region, site group, tenant and status."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("region", mcp.Description("filter by region id or slug (includes sites in descendant regions)")),
		mcp.WithString("group", mcp.Description("filter by site group id or slug")),
		mcp.WithString("tenant", mcp.Description("filter by tenant id or slug")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, planned, staging, decommissioning, retired")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.SiteFilter{
			Search: req.GetString("search", ""),
			Region: req.GetString("region", ""),
			Group:  req.GetString("group", ""),
			Tenant: req.GetString("tenant", ""),
			Status: req.GetString("status", ""),
			Tag:    req.GetString("tag", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListSites(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_site",
		mcp.WithDescription("Get one site by numeric id or slug, with all fields including custom_fields, tags and comments."),
		mcp.WithString("site", mcp.Required(), mcp.Description("site id or slug, e.g. 12 or ams-dc1")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("site")
		if err != nil {
			return errResult(err)
		}
		site, err := b.GetSite(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(site)
	})

	add(mcp.NewTool("list_locations",
		mcp.WithDescription("List locations — the nested subdivisions of a site (buildings, floors, rooms, cages) that racks sit in."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("parent", mcp.Description(parentDesc)),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.LocationFilter{
			Search: req.GetString("search", ""),
			Site:   req.GetString("site", ""),
			Parent: req.GetString("parent", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListLocations(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_location",
		mcp.WithDescription("Get one location by numeric id or slug, with all fields including site, parent, rack and device counts."),
		mcp.WithString("location", mcp.Required(), mcp.Description("location id or slug, e.g. 7 or ams-dc1-floor2")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("location")
		if err != nil {
			return errResult(err)
		}
		loc, err := b.GetLocation(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(loc)
	})

	add(mcp.NewTool("list_racks",
		mcp.WithDescription("List racks, optionally within one site or location."),
		mcp.WithString("search", mcp.Description("free-text filter")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("location", mcp.Description("filter by location id or slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.RackFilter{
			Search:   req.GetString("search", ""),
			Site:     req.GetString("site", ""),
			Location: req.GetString("location", ""),
			Brief:    req.GetBool("brief", false),
		}
		res, err := b.ListRacks(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("list_devices",
		mcp.WithDescription("List devices, filterable by region, site, location, rack, role, tenant, status and tag."),
		mcp.WithString("search", mcp.Description("free-text filter (name, serial, asset tag)")),
		mcp.WithString("region", mcp.Description("filter by region id or slug")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("location", mcp.Description("filter by location id or slug")),
		mcp.WithString("rack", mcp.Description("filter by rack id or name")),
		mcp.WithString("role", mcp.Description("filter by device role id or slug")),
		mcp.WithString("tenant", mcp.Description("filter by tenant id or slug")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, offline, planned, staged, failed, decommissioning")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.DeviceFilter{
			Search:   req.GetString("search", ""),
			Region:   req.GetString("region", ""),
			Site:     req.GetString("site", ""),
			Location: req.GetString("location", ""),
			Rack:     req.GetString("rack", ""),
			Role:     req.GetString("role", ""),
			Tenant:   req.GetString("tenant", ""),
			Status:   req.GetString("status", ""),
			Tag:      req.GetString("tag", ""),
			Brief:    req.GetBool("brief", false),
		}
		res, err := b.ListDevices(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_device",
		mcp.WithDescription("Get one device by numeric id or exact name, with all fields including custom_fields, tags and comments."),
		mcp.WithString("device", mcp.Required(), mcp.Description("device id or exact name, e.g. 42 or sw-core-01")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("device")
		if err != nil {
			return errResult(err)
		}
		dev, err := b.GetDevice(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(dev)
	})

	add(mcp.NewTool("list_device_interfaces",
		mcp.WithDescription("List the network interfaces of a device."),
		mcp.WithString("device", mcp.Required(), mcp.Description("device id or exact name")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		device, err := req.RequireString("device")
		if err != nil {
			return errResult(err)
		}
		f := netboxx.DeviceInterfaceFilter{Device: device, Brief: req.GetBool("brief", false)}
		res, err := b.ListDeviceInterfaces(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	return names
}
