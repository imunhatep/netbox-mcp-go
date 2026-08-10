package command

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
	"github.com/imunhatep/netbox-mcp-go/internal/mcpserver"
	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

// ServeCommand runs the MCP HTTP server.
type ServeCommand struct{}

func (c ServeCommand) Command() *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "run the NetBox MCP HTTP server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "netbox-url",
				Usage:    "base URL of the NetBox instance",
				Sources:  cli.EnvVars("NETBOX_URL"),
				Required: true,
			},
			&cli.StringFlag{
				Name: "netbox-token",
				Usage: "NetBox API token; required unless --token-passthrough is set, " +
					"in which case it is the fallback for requests without one",
				Sources: cli.EnvVars("NETBOX_TOKEN"),
			},
			&cli.BoolFlag{
				Name: "token-passthrough",
				Usage: "authenticate to NetBox with the token from each request's " +
					"Authorization: Bearer header, so callers use their own NetBox permissions",
				Sources: cli.EnvVars("NETBOX_TOKEN_PASSTHROUGH"),
			},
			&cli.StringFlag{
				Name:    "listen",
				Usage:   "HTTP listen address",
				Value:   "127.0.0.1:3031",
				Sources: cli.EnvVars("NETBOX_MCP_LISTEN"),
			},
			&cli.DurationFlag{
				Name:    "request-timeout",
				Usage:   "per-request timeout to NetBox",
				Value:   30 * time.Second,
				Sources: cli.EnvVars("NETBOX_MCP_REQUEST_TIMEOUT"),
			},
			&cli.IntFlag{
				Name:    "page-size",
				Usage:   "objects returned per page by list tools",
				Value:   50,
				Sources: cli.EnvVars("NETBOX_MCP_PAGE_SIZE"),
			},
		},
		Action: c.run,
	}
}

func (c ServeCommand) run(ctx context.Context, cmd *cli.Command) error {
	cfg := config.FromCommand(cmd)
	if err := cfg.Validate(); err != nil {
		return err
	}
	svc, err := netboxx.NewService(cfg)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return mcpserver.Run(ctx, cfg, svc)
}
