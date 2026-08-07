// Package internal wires the CLI application and logging.
package internal

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/imunhatep/netbox-mcp-go/internal/version"
)

// NewApp builds the root CLI command. Subcommands are attached by main.
func NewApp() *cli.Command {
	return &cli.Command{
		Name:                  "netbox-mcp",
		Usage:                 "MCP server exposing a NetBox instance to Claude Code",
		UsageText:             "netbox-mcp [global options] command [command options]",
		Description:           "Reads sites, racks, devices, interfaces, prefixes, IP addresses, VLANs, VRFs, clusters, virtual machines and tenants from NetBox.",
		Version:               version.Version,
		EnableShellCompletion: true,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "log verbosity: 0=fatal 1=error 2=warn 3=info 4=debug 5=trace",
				Value:   3,
				Sources: cli.EnvVars("NETBOX_MCP_VERBOSE"),
			},
			&cli.StringFlag{
				Name:    "log-name",
				Usage:   "identifier tagged as the `service` field on every log line; set a distinct value per server to tell them apart when several run in one pod",
				Value:   "netbox-mcp",
				Sources: cli.EnvVars("NETBOX_MCP_LOG_NAME"),
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			SetupLogging(int(cmd.Int("verbose")), cmd.String("log-name"))
			return ctx, nil
		},
	}
}

// SetupLogging configures the global zerolog logger, its level, and a
// persistent "service" field set to name. Tagging every line with the server's
// name lets logs be told apart when several MCP servers share a pod's stderr.
func SetupLogging(level int, name string) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.DateTime}).
		With().Str("service", name).Logger()

	switch level {
	case 0:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case 1:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case 2:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case 3:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case 4:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}
}
