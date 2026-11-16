// Package skills ships the Claude Code skill that teaches a caller how to use
// netbox-mcp. It is embedded in the binary so the skill always matches the
// server: connecting clients get it through the get_skill tool, and anyone who
// only runs the binary or the container image can install it by hand:
//
//	netbox-mcp skill --install
//	podman run --rm ghcr.io/imunhatep/netbox-mcp-go:latest skill \
//	  > ~/.claude/skills/using-netbox-mcp/SKILL.md
//
// The skill is edited here, in skills/using-netbox-mcp/SKILL.md. The
// {{VERSION}} placeholder is filled in by internal/skill.
package skills

import _ "embed"

// UsingNetboxMCPName is the skill's directory name under ~/.claude/skills.
const UsingNetboxMCPName = "using-netbox-mcp"

// UsingNetboxMCP is the SKILL.md template, version-locked to this binary.
//
//go:embed using-netbox-mcp/SKILL.md
var UsingNetboxMCP string
