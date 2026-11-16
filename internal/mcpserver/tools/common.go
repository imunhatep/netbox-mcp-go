// Package tools registers MCP tools backed by the NetBox service.
package tools

import (
	"encoding/json"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("failed to encode result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errResult(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

// optList reads an optional list argument. Models sometimes send a
// comma-separated string where an array is declared, so both are accepted.
func optList(req mcp.CallToolRequest, name string) []string {
	if s, ok := req.GetArguments()[name].(string); ok {
		var out []string
		for _, v := range strings.Split(s, ",") {
			if v = strings.TrimSpace(v); v != "" {
				out = append(out, v)
			}
		}
		return out
	}
	return req.GetStringSlice(name, nil)
}

// fieldsDesc documents the "fields" argument shared by the list tools.
const fieldsDesc = "top-level fields to return per object, e.g. [\"name\",\"status\",\"site\",\"primary_ip4\",\"custom_fields\"]; " +
	"id is always included and this overrides brief. Nested paths such as site.name are not supported; " +
	"related objects come back in their brief form. Use it to get just the attributes you need for many objects in one call."

// optPage reads an optional 1-based "page" argument, defaulting to 1.
func optPage(req mcp.CallToolRequest) int {
	p := req.GetInt("page", 1)
	if p < 1 {
		return 1
	}
	return p
}
