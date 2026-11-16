package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/imunhatep/netbox-mcp-go/internal/skill"
)

// RegisterSkill adds get_skill, which hands out the agent skill embedded in
// this binary, and returns its name. It makes no NetBox call, so it needs no
// token even with --token-passthrough.
func RegisterSkill(s *server.MCPServer, serverVersion string) []string {
	t := mcp.NewTool("get_skill",
		mcp.WithDescription("The agent skill \""+skill.Name+"\" (SKILL.md) that explains how to use this server, "+
			"matched to the server version. Pass installed_version (from the netbox-skill-version line of the "+
			"local copy) to learn whether it is current; content is returned when it is not, or always with "+
			"force=true. Write content verbatim to install_path. Makes no NetBox call."),
		mcp.WithString("installed_version", mcp.Description("version from the netbox-skill-version line of the installed SKILL.md; omit when it is not installed")),
		mcp.WithBoolean("force", mcp.Description("return content even when installed_version is current")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
	s.AddTool(t, func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return jsonResult(skill.Check(serverVersion, req.GetString("installed_version", ""), req.GetBool("force", false)))
	})
	return []string{t.Name}
}
