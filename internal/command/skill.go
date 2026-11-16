package command

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/imunhatep/netbox-mcp-go/internal/skill"
	"github.com/imunhatep/netbox-mcp-go/internal/version"
)

// SkillCommand prints, checks or installs the embedded Claude Code skill, so
// users who only run the binary or the image can install it without cloning
// the repository. Connected clients are kept current by the server itself
// (instructions + get_skill); this is the manual path.
type SkillCommand struct{}

// Command returns the "skill" subcommand.
func (c SkillCommand) Command() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "print the " + skill.Name + " agent skill (SKILL.md) for this version; --install writes it, --check compares",
		Description: "Install it with:\n\n" +
			"   netbox-mcp skill --install\n\n" +
			"or, from the image:\n\n" +
			"   mkdir -p ~/.claude/skills/" + skill.Name + "\n" +
			"   podman run --rm ghcr.io/imunhatep/netbox-mcp-go:latest skill \\\n" +
			"     > ~/.claude/skills/" + skill.Name + "/SKILL.md",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "install", Usage: "write the skill into --dir (only when the installed version differs)"},
			&cli.BoolFlag{Name: "check", Usage: "report the installed and the current version"},
			&cli.StringFlag{Name: "dir", Value: skill.DefaultDir(), Usage: "skills directory"},
		},
		Action: c.run,
	}
}

func (c SkillCommand) run(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	content := skill.Content(version.Version)
	current := skill.InstalledVersion(content)
	dir := cmd.String("dir")

	switch {
	case cmd.Bool("check"):
		path, installed := skill.Path(dir), ""
		if b, err := os.ReadFile(path); err == nil {
			installed = skill.InstalledVersion(string(b))
		}
		state := "current"
		switch {
		case installed == "":
			state = "not installed"
		case installed != current:
			state = "outdated"
		}
		_, err := fmt.Fprintf(w, "%s: %s (installed %q, current %q)\n", path, state, installed, current)
		return err
	case cmd.Bool("install"):
		path, previous, written, err := skill.Install(dir, content)
		switch {
		case err != nil:
			return err
		case !written:
			_, err = fmt.Fprintf(w, "%s is current (%s)\n", path, current)
		case previous == "":
			_, err = fmt.Fprintf(w, "installed %s (%s)\n", path, current)
		default:
			_, err = fmt.Fprintf(w, "updated %s: %s -> %s\n", path, previous, current)
		}
		return err
	default:
		_, err := fmt.Fprint(w, content)
		return err
	}
}
