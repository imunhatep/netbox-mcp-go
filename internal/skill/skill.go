// Package skill holds the agent skill that explains how to use the netbox-mcp
// server. The text lives in skills/using-netbox-mcp and is compiled into the
// binary, so the skill always matches the server: the MCP server hands it out
// (get_skill tool, announced in the server instructions) and the CLI installs it.
package skill

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/imunhatep/netbox-mcp-go/skills"
)

// Name is the skill's directory and frontmatter name.
const Name = skills.UsingNetboxMCPName

// InstallPath is where clients are told to install the skill. It is a
// constant on purpose: the server never lets a request choose where a client
// writes.
const InstallPath = "~/.claude/skills/" + Name + "/SKILL.md"

// template is SKILL.md with a {{VERSION}} placeholder.
var template = skills.UsingNetboxMCP

// Version identifies the skill text: the server version plus a hash of the
// text, so it changes with every release and whenever the skill changes, even
// without a release.
func Version(serverVersion string) string {
	sum := sha256.Sum256([]byte(template))
	return serverVersion + "+" + hex.EncodeToString(sum[:4])
}

// Content returns SKILL.md with its version line filled in.
func Content(serverVersion string) string {
	return strings.ReplaceAll(template, "{{VERSION}}", Version(serverVersion))
}

var versionLine = regexp.MustCompile(`netbox-skill-version:\s*(\S+)\s*-->`)

// InstalledVersion extracts the version from a SKILL.md, or "" when there is none.
func InstalledVersion(text string) string {
	if m := versionLine.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	return ""
}

// DefaultDir is the personal Claude Code skills directory, ~/.claude/skills.
func DefaultDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".claude", "skills")
	}
	return filepath.Join(home, ".claude", "skills")
}

// Path is where the skill lives under a skills directory.
func Path(dir string) string { return filepath.Join(dir, Name, "SKILL.md") }

// Install writes content (a rendered SKILL.md) under dir unless the installed
// copy already has the same version. It returns the path, the previously
// installed version ("" if none) and whether the file was written.
func Install(dir, content string) (path, previous string, written bool, err error) {
	path = Path(dir)
	if b, err := os.ReadFile(path); err == nil {
		previous = InstalledVersion(string(b))
		if previous != "" && previous == InstalledVersion(content) {
			return path, previous, false, nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, previous, false, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return path, previous, false, fmt.Errorf("write %s: %w", path, err)
	}
	return path, previous, true, nil
}

// Answer is the get_skill tool result.
type Answer struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	InstalledVersion string `json:"installed_version,omitempty"`
	UpToDate         bool   `json:"up_to_date"`
	InstallPath      string `json:"install_path"`
	Instructions     string `json:"instructions"`
	Content          string `json:"content,omitempty"`
}

// Check compares an installed version against this server's skill. Content is
// included only when the installed copy is missing or outdated, or force is set.
func Check(serverVersion, installed string, force bool) Answer {
	a := Answer{
		Name:             Name,
		Version:          Version(serverVersion),
		InstalledVersion: installed,
		InstallPath:      InstallPath,
	}
	a.UpToDate = installed == a.Version
	switch {
	case a.UpToDate && !force:
		a.Instructions = "The installed skill is current; nothing to do."
	case installed == "":
		a.Instructions = "Create the directory and write content verbatim to install_path, then tell the user the skill was installed."
	default:
		a.Instructions = "Overwrite install_path with content verbatim, then tell the user the skill was updated from installed_version to version."
	}
	if !a.UpToDate || force {
		a.Content = Content(serverVersion)
	}
	return a
}
