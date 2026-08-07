package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type TenancyBackend interface {
	ListTenantGroups(ctx context.Context, search, parent string, brief bool, page int) (*netboxx.Page, error)
	ListTenants(ctx context.Context, search, group string, brief bool, page int) (*netboxx.Page, error)
	GetTenant(ctx context.Context, ref string) (netboxx.Object, error)
	ListContacts(ctx context.Context, search, group string, brief bool, page int) (*netboxx.Page, error)
	Status(ctx context.Context) (netboxx.Object, error)
}

// RegisterTenancy adds tenant group, tenant, contact and instance-status tools
// and returns their names.
func RegisterTenancy(s *server.MCPServer, b TenancyBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_tenant_groups",
		mcp.WithDescription("List tenant groups — the hierarchy that tenants are organized into (e.g. by business unit)."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("parent", mcp.Description(parentDesc)),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.ListTenantGroups(ctx, req.GetString("search", ""), req.GetString("parent", ""),
			req.GetBool("brief", false), optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("list_tenants",
		mcp.WithDescription("List tenants (organizational owners of resources), optionally within one tenant group."),
		mcp.WithString("search", mcp.Description("free-text filter (name, slug, description)")),
		mcp.WithString("group", mcp.Description("filter by tenant group id or slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.ListTenants(ctx, req.GetString("search", ""), req.GetString("group", ""),
			req.GetBool("brief", false), optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("get_tenant",
		mcp.WithDescription("Get one tenant by numeric id or slug, with all fields including group, custom_fields, tags and comments."),
		mcp.WithString("tenant", mcp.Required(), mcp.Description("tenant id or slug, e.g. 4 or acme-corp")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ref, err := req.RequireString("tenant")
		if err != nil {
			return errResult(err)
		}
		tenant, err := b.GetTenant(ctx, ref)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(tenant)
	})

	add(mcp.NewTool("list_contacts",
		mcp.WithDescription("List contacts — the people or teams that can be assigned to sites, devices, tenants and other objects."),
		mcp.WithString("search", mcp.Description("free-text filter (name, title, email, phone)")),
		mcp.WithString("group", mcp.Description("filter by contact group id or slug")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.ListContacts(ctx, req.GetString("search", ""), req.GetString("group", ""),
			req.GetBool("brief", false), optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	add(mcp.NewTool("netbox_status",
		mcp.WithDescription("Get NetBox instance status: version and installed plugins."),
	), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		st, err := b.Status(ctx)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(st)
	})

	return names
}
