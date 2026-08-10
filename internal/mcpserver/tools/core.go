package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type CoreBackend interface {
	ListChangelog(ctx context.Context, f netboxx.ChangelogFilter, page int) (*netboxx.Page, error)
}

// RegisterCore adds the changelog (audit trail) tool and returns its name.
func RegisterCore(s *server.MCPServer, b CoreBackend) []string {
	var names []string
	add := func(t mcp.Tool, h server.ToolHandlerFunc) {
		s.AddTool(t, h)
		names = append(names, t.Name)
	}

	add(mcp.NewTool("list_changelog",
		mcp.WithDescription("List NetBox changelog records — the audit trail of who created, updated or "+
			"deleted an object and when. Each record includes prechange_data and postchange_data "+
			"snapshots, so this answers questions like \"who decommissioned this device\", \"what "+
			"changed at this site last week\" or \"why is this prefix marked reserved\". To see the "+
			"history of one object, pass its type and id (object_type=dcim.device, object_id=42)."),
		mcp.WithString("search", mcp.Description("free-text filter (matches the object representation and user name)")),
		mcp.WithString("user", mcp.Description("filter by acting user id or username")),
		mcp.WithString("action", mcp.Description("filter by action: created, updated or deleted")),
		mcp.WithString("object_type", mcp.Description("filter by changed object type as a NetBox app.model label, e.g. dcim.device, ipam.prefix, virtualization.virtualmachine")),
		mcp.WithString("object_id", mcp.Description("filter by the changed object's numeric id; use together with object_type")),
		mcp.WithString("request_id", mcp.Description("filter by request id — groups every change made by a single request, e.g. one bulk edit or script run")),
		mcp.WithString("after", mcp.Description("only changes at or after this time, e.g. 2026-08-01 or 2026-08-01T12:30:00Z")),
		mcp.WithString("before", mcp.Description("only changes at or before this time")),
		mcp.WithBoolean("brief", mcp.Description(briefDesc)),
		mcp.WithNumber("page", mcp.Description("1-based page number; records are newest first")),
	), func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		f := netboxx.ChangelogFilter{
			Search:     req.GetString("search", ""),
			User:       req.GetString("user", ""),
			Action:     req.GetString("action", ""),
			ObjectType: req.GetString("object_type", ""),
			ObjectID:   req.GetString("object_id", ""),
			RequestID:  req.GetString("request_id", ""),
			After:      req.GetString("after", ""),
			Before:     req.GetString("before", ""),
			Brief:      req.GetBool("brief", false),
		}
		res, err := b.ListChangelog(ctx, f, optPage(req))
		if err != nil {
			return errResult(err)
		}
		return jsonResult(res)
	})

	return names
}
