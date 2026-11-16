package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

type SearchBackend interface {
	Search(ctx context.Context, f netboxx.SearchFilter) ([]netboxx.SearchGroup, error)
}

// RegisterSearch adds the global search tool and returns its name.
func RegisterSearch(s *server.MCPServer, b SearchBackend) []string {
	t := mcp.NewTool("search",
		mcp.WithDescription("Global free-text search across NetBox: regions, sites, locations, racks, devices, "+
			"prefixes, IP addresses, VLANs, VRFs, clusters, virtual machines, tenants and contacts. Returns brief "+
			"matches grouped by object type with the total match count per type; follow up with the matching "+
			"get_*/list_* tool for full details. An IP or CIDR query only searches prefixes and IP addresses; "+
			"to resolve specific addresses use lookup_ip_addresses instead."),
		mcp.WithString("query", mcp.Required(), mcp.Description("free-text search query, e.g. a hostname, IP or tag value")),
		mcp.WithArray("types", mcp.WithStringItems(),
			mcp.Description("only search these object types: "+strings.Join(netboxx.SearchTypes(), ", "))),
		mcp.WithNumber("limit", mcp.Description("max results returned per object type (default 10)")),
	)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return errResult(err)
		}
		f := netboxx.SearchFilter{
			Query:   query,
			Types:   optList(req, "types"),
			PerType: req.GetInt("limit", 10),
		}
		groups, err := b.Search(ctx, f)
		if err != nil {
			return errResult(err)
		}
		return jsonResult(groups)
	})
	return []string{t.Name}
}
