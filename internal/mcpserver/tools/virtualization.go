package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type VirtualizationBackend interface {
	ListClusters(ctx context.Context, search, site string, brief bool, page int) (*netboxx.Page, error)
	ListVirtualMachines(ctx context.Context, f netboxx.VMFilter, page int) (*netboxx.Page, error)
	GetVirtualMachine(ctx context.Context, ref string) (netboxx.Object, error)
	ListVMInterfaces(ctx context.Context, vm string, brief bool, page int) (*netboxx.Page, error)
}

// RegisterVirtualization adds cluster and virtual machine tools and returns their names.
func RegisterVirtualization(s *server.MCPServer, b VirtualizationBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_clusters",
		mcp.WithDescription("List virtualization clusters."),
		mcp.WithString("search", mcp.Description("free-text filter (name, description)")),
		mcp.WithString("site", mcp.Description("filter by site id or slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.ListClusters(ctx, req.GetString("search", ""), req.GetString("site", ""),
			req.GetBool("brief", false), optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("list_virtual_machines",
		mcp.WithDescription("List virtual machines, filterable by cluster, site, tenant, status and tag."),
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
			Brief:   req.GetBool("brief", false),
		}
		res, err := b.ListVirtualMachines(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_virtual_machine",
		mcp.WithDescription("Get one virtual machine by numeric id or exact name, with all fields including custom_fields, tags and comments."),
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
		res, err := b.ListVMInterfaces(ctx, vm, req.GetBool("brief", false), optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	return names
}
