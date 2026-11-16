package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type VirtualizationBackend interface {
	ListClusters(ctx context.Context, f netboxx.ClusterFilter, page int) (*netboxx.Page, error)
	ListVirtualMachines(ctx context.Context, f netboxx.VMFilter, page int) (*netboxx.Page, error)
	GetVirtualMachine(ctx context.Context, ref string) (netboxx.Object, error)
	ListVMInterfaces(ctx context.Context, f netboxx.VMInterfaceFilter, page int) (*netboxx.Page, error)
}

// RegisterVirtualization adds cluster and virtual machine tools and returns their names.
func RegisterVirtualization(s *server.MCPServer, b VirtualizationBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_clusters",
		mcp.WithDescription("List virtualization clusters, filterable by type, group, site, tenant, status and tag. "+
			"There can be hundreds of clusters: always filter (or search) and use brief or fields rather than paging through full objects."),
		mcp.WithString("search", mcp.Description("free-text filter (name, description)")),
		mcp.WithString("type", mcp.Description("filter by cluster type id or slug, e.g. vmware, kubernetes")),
		mcp.WithString("group", mcp.Description("filter by cluster group id or slug")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("tenant", mcp.Description("filter by tenant id or slug")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, planned, staged, offline, decommissioning")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithArray("fields", mcp.WithStringItems(), mcp.Description(fieldsDesc)),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.ClusterFilter{
			Search: req.GetString("search", ""),
			Type:   req.GetString("type", ""),
			Group:  req.GetString("group", ""),
			Site:   req.GetString("site", ""),
			Tenant: req.GetString("tenant", ""),
			Status: req.GetString("status", ""),
			Tag:    req.GetString("tag", ""),
			Fields: optList(req, "fields"),
			Brief:  req.GetBool("brief", false),
		}
		res, err := b.ListClusters(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("list_virtual_machines",
		mcp.WithDescription("List virtual machines, filterable by cluster, site, tenant, status and tag. "+
			"To fetch several known VMs, pass names=[...] or ids=[...] in ONE call instead of calling get_virtual_machine per VM. "+
			"Use fields=[...] to get just the attributes you need for many VMs; use brief only to discover names."),
		mcp.WithArray("names", mcp.WithStringItems(), mcp.Description("exact VM names to fetch together; a name can repeat across clusters")),
		mcp.WithArray("ids", mcp.WithStringItems(), mcp.Description("numeric VM ids to fetch together")),
		mcp.WithArray("fields", mcp.WithStringItems(), mcp.Description(fieldsDesc)),
		mcp.WithString("search", mcp.Description("free-text filter (name, description)")),
		mcp.WithString("cluster", mcp.Description("filter by cluster id or name")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithString("tenant", mcp.Description("filter by tenant id or slug")),
		mcp.WithString("status", mcp.Description("filter by status, e.g. active, offline, planned, staged, failed, decommissioning")),
		mcp.WithString("tag", mcp.Description("filter by tag slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.VMFilter{
			Search:  req.GetString("search", ""),
			Cluster: req.GetString("cluster", ""),
			Site:    req.GetString("site", ""),
			Tenant:  req.GetString("tenant", ""),
			Status:  req.GetString("status", ""),
			Tag:     req.GetString("tag", ""),
			IDs:     optList(req, "ids"),
			Names:   optList(req, "names"),
			Fields:  optList(req, "fields"),
			Brief:   req.GetBool("brief", false),
		}
		res, err := b.ListVirtualMachines(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_virtual_machine",
		mcp.WithDescription("Get one virtual machine by numeric id or exact name, with all fields including custom_fields, tags and comments. "+
			"For more than one VM call list_virtual_machines once with names=[...] or ids=[...]; do not call get_virtual_machine in a loop."),
		mcp.WithString("vm", mcp.Required(), mcp.Description("VM id or exact name, e.g. 7 or web-prod-01")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("vm")
		if err != nil {
			return errResult(err)
		}
		vm, err := b.GetVirtualMachine(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(vm)
	})

	add(mcp.NewTool("list_vm_interfaces",
		mcp.WithDescription("List the network interfaces of a virtual machine."),
		mcp.WithString("vm", mcp.Required(), mcp.Description("VM id or exact name")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vm, err := req.RequireString("vm")
		if err != nil {
			return errResult(err)
		}
		f := netboxx.VMInterfaceFilter{VM: vm, Brief: req.GetBool("brief", false)}
		res, err := b.ListVMInterfaces(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	return names
}
