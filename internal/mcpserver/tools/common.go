// Package tools registers MCP tools backed by the NetBox service.
package tools

import (
	"encoding/json"

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

// optPage reads an optional 1-based "page" argument, defaulting to 1.
func optPage(req mcp.CallToolRequest) int {
	p := req.GetInt("page", 1)
	if p < 1 {
		return 1
	}
	return p
}
