// Package config holds server configuration and its validation.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
)

// Config is the fully-resolved server configuration.
type Config struct {
	NetBoxURL      string
	NetBoxToken    string
	Listen         string
	RequestTimeout time.Duration
	PageSize       int
	Verbose        int

	// TokenPassthrough authenticates each MCP request to NetBox with the
	// token from its own Authorization: Bearer header instead of NetBoxToken.
	// It makes NetBoxToken optional (it becomes the fallback for requests
	// that carry no header).
	TokenPassthrough bool
}

// FromCommand builds a Config from the serve command's flags.
func FromCommand(cmd *cli.Command) *Config {
	base := strings.TrimRight(cmd.String("netbox-url"), "/")
	// Accept URLs pasted with the API suffix; the client appends /api itself.
	base = strings.TrimSuffix(base, "/api")
	return &Config{
		NetBoxURL:      base,
		NetBoxToken:    cmd.String("netbox-token"),
		Listen:         cmd.String("listen"),
		RequestTimeout: cmd.Duration("request-timeout"),
		PageSize:       int(cmd.Int("page-size")),
		Verbose:        int(cmd.Int("verbose")),

		TokenPassthrough: cmd.Bool("token-passthrough"),
	}
}

// Validate fails fast on missing or malformed required settings.
func (c *Config) Validate() error {
	if c.NetBoxURL == "" {
		return errors.New("NETBOX_URL (--netbox-url) is required")
	}
	// With passthrough the token arrives per request, so a configured one is
	// only an optional fallback.
	if c.NetBoxToken == "" && !c.TokenPassthrough {
		return errors.New("NETBOX_TOKEN (--netbox-token) is required " +
			"unless --token-passthrough is set")
	}
	u, err := url.Parse(c.NetBoxURL)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("NETBOX_URL must be an absolute URL, got %q", c.NetBoxURL)
	}
	if c.PageSize < 1 || c.PageSize > 1000 {
		return errors.New("--page-size must be between 1 and 1000")
	}
	return nil
}
