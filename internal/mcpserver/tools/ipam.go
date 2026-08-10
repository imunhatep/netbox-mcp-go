package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type IPAMBackend interface {
	ListPrefixes(ctx context.Context, f netboxx.PrefixFilter, page int) (*netboxx.Page, error)
	GetPrefix(ctx context.Context, ref string) (netboxx.Object, error)
	ListIPAddresses(ctx context.Context, f netboxx.IPFilter, page int) (*netboxx.Page, error)
	GetIPAddress(ctx context.Context, ref string) (netboxx.Object, error)
	ListVLANs(ctx context.Context, f netboxx.VLANFilter, page int) (*netboxx.Page, error)
	ListVRFs(ctx context.Context, f netboxx.VRFFilter, page int) (*netboxx.Page, error)
}

// RegisterIPAM adds prefix, IP address, VLAN and VRF tools and returns their names.
func RegisterIPAM(s *server.MCPServer, b IPAMBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_prefixes",
		mcp.WithDescription("List IP prefixes (subnets), filterable by parent prefix, site, VRF, status, address family and tag."),
		mcp.WithString("search", mcp.Description("free-text filter (prefix, description)")),
		mcp.WithString("within", mcp.Description("only prefixes contained in this CIDR (inclusive), e.g. 10.0.0.0/8")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("vrf", mcp.Description("filter by VRF id or name")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, container, reserved, deprecated")),
		mcp.WithString("family", mcp.Description("IP family: 4 or 6")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.PrefixFilter{
			Search: req.GetString("search", ""),
			Within: req.GetString("within", ""),
			Site:   req.GetString("site", ""),
			VRF:    req.GetString("vrf", ""),
			Status: req.GetString("status", ""),
			Family: req.GetString("family", ""),
			Tag:    req.GetString("tag", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListPrefixes(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_prefix",
		mcp.WithDescription("Get one prefix by numeric id or exact CIDR, with all fields including custom_fields, tags and comments."),
		mcp.WithString("prefix", mcp.Required(), mcp.Description("prefix id or CIDR, e.g. 815 or 10.1.0.0/24")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("prefix")
		if err != nil {
			return errResult(err)
		}
		p, err := b.GetPrefix(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(p)
	})

	add(mcp.NewTool("list_ip_addresses",
		mcp.WithDescription("List IP addresses, filterable by containing prefix, device, virtual machine, VRF, status and tag."),
		mcp.WithString("search", mcp.Description("free-text filter (address, DNS name, description)")),
		mcp.WithString("parent", mcp.Description("only IPs within this prefix, e.g. 10.1.0.0/24")),
		mcp.WithString("device", mcp.Description("only IPs assigned to this device (id or name)")),
		mcp.WithString("vm", mcp.Description("only IPs assigned to this virtual machine (id or name)")),
		mcp.WithString("vrf", mcp.Description("filter by VRF id or name")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, reserved, deprecated, dhcp, slaac")),
		mcp.WithString("dns_name", mcp.Description("filter by exact DNS name")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.IPFilter{
			Search:  req.GetString("search", ""),
			Parent:  req.GetString("parent", ""),
			Device:  req.GetString("device", ""),
			VM:      req.GetString("vm", ""),
			VRF:     req.GetString("vrf", ""),
			Status:  req.GetString("status", ""),
			DNSName: req.GetString("dns_name", ""),
			Tag:     req.GetString("tag", ""),
			Brief:   req.GetBool("brief", false),
		}
		res, err := b.ListIPAddresses(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_ip_address",
		mcp.WithDescription("Get one IP address by numeric id or address (with or without mask), with all fields including custom_fields and tags."),
		mcp.WithString("ip", mcp.Required(), mcp.Description("IP id or address, e.g. 101, 10.1.0.5 or 10.1.0.5/24")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("ip")
		if err != nil {
			return errResult(err)
		}
		ip, err := b.GetIPAddress(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(ip)
	})

	add(mcp.NewTool("list_vlans",
		mcp.WithDescription("List VLANs, filterable by site and 802.1Q id."),
		mcp.WithString("search", mcp.Description("free-text filter (name, description)")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("vid", mcp.Description("filter by 802.1Q VLAN id, e.g. 120")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.VLANFilter{
			Search: req.GetString("search", ""),
			Site:   req.GetString("site", ""),
			VID:    req.GetString("vid", ""),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListVLANs(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("list_vrfs",
		mcp.WithDescription("List VRFs (virtual routing and forwarding tables)."),
		mcp.WithString("search", mcp.Description("free-text filter (name, route distinguisher)")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.VRFFilter{Search: req.GetString("search", ""), Brief: req.GetBool("brief", false)}
		res, err := b.ListVRFs(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	return names
}
