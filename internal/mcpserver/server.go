// Package mcpserver builds and runs the MCP HTTP server.
package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
	"github.com/imunhatep/netbox-mcp-go/internal/mcpserver/tools"
	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
	"github.com/imunhatep/netbox-mcp-go/internal/skill"
	"github.com/imunhatep/netbox-mcp-go/internal/version"
)

// instructions are sent on connect. They carry the skill version, so a
// connecting agent can tell whether its installed copy of the skill is current
// and update it — every release thereby ships its matching skill.
func instructions(serverVersion string) string {
	v := skill.Version(serverVersion)
	return `netbox-mcp: read-only access to NetBox (sites, racks, devices, interfaces, prefixes, IP addresses,
VLANs, VRFs, clusters, virtual machines, tenants, contacts, changelog). No tool writes to NetBox.
Batch lookups: lookup_ip_addresses for many IPs, list_devices / list_virtual_machines with names or ids
for many hosts, fields=[...] to return only the attributes you need. Do not loop get_* tools.

Skill: the agent skill "` + skill.Name + `" (current version ` + v + `) explains how to pick tools,
the id-or-slug rules, batching, paging, brief and fields, and how to read the results. Keep it
installed and current: on connect, and whenever the user asks to install or update the netbox skill,
read ` + skill.InstallPath + `. If the file is missing or does not contain the line
"netbox-skill-version: ` + v + ` -->", call get_skill (installed_version = the version found in the
file, if any) and write the returned "content" verbatim to that one path, creating the directory.
Write nothing else. Then tell the user the skill was installed or updated, from which version to
which. Do not edit the file by hand; the next update overwrites it.`
}

// New builds the MCP server and registers every tool group. All tools are
// read-only: the server never writes to NetBox.
func New(cfg *config.Config, svc *netboxx.Service) *server.MCPServer {
	s := server.NewMCPServer(
		"netbox-mcp",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithHooks((&mcpLogger{}).hooks()),
		server.WithInstructions(instructions(version.Version)),
	)
	tools.RegisterSkill(s, version.Version)
	tools.RegisterSearch(s, svc)
	tools.RegisterDCIM(s, svc)
	tools.RegisterIPAM(s, svc)
	tools.RegisterVirtualization(s, svc)
	tools.RegisterTenancy(s, svc)
	tools.RegisterCore(s, svc)
	return s
}

const (
	// heartbeatInterval is how often an idle listening stream (GET /mcp, or
	// the legacy /sse stream) carries a ping, well inside client idle limits.
	heartbeatInterval = 30 * time.Second
	// sessionIdleTTL releases the state of sessions that saw no request and
	// no heartbeat for this long.
	sessionIdleTTL = 30 * time.Minute
)

// newMux builds the HTTP routes. A single MCPServer instance backs both
// transports:
//   - Streamable HTTP (the current MCP spec) at /mcp
//   - HTTP+SSE (the 2024-11-05 legacy transport) with the event stream at
//     /sse and client->server messages at /message
//
// SSE is mounted for clients that only speak the older transport. Both share
// the same tool registry, so every tool behaves identically regardless of
// which transport a client connects with.
func newMux(cfg *config.Config, svc *netboxx.Service) *http.ServeMux {
	mcp := New(cfg, svc)

	// Claude Code drops a GET /mcp listening stream that has carried nothing
	// for ~6 minutes, and after three such drops in a row it discards the
	// session and re-initializes. The heartbeat pings on that stream (each
	// ping resets the client's error count), which stopped a loop of ~370
	// re-initializations a day. Sessions whose client goes away without a
	// DELETE are otherwise never released; the idle TTL sweeps them, and the
	// heartbeat keeps live ones from counting as idle.
	httpOpts := []server.StreamableHTTPOption{
		server.WithHeartbeatInterval(heartbeatInterval),
		server.WithSessionIdleTTL(sessionIdleTTL),
	}
	sseOpts := []server.SSEOption{
		server.WithKeepAlive(true),
		server.WithKeepAliveInterval(heartbeatInterval),
	}

	// With passthrough enabled, each transport lifts the request's bearer
	// token into the handler context so NetBox is queried as the caller.
	if cfg.TokenPassthrough {
		httpOpts = append(httpOpts, server.WithHTTPContextFunc(tokenPassthroughContext))
		sseOpts = append(sseOpts, server.WithSSEContextFunc(tokenPassthroughContext))
	}

	mux := http.NewServeMux()
	// Each transport is wrapped so the HTTP layer (client IP, method, path,
	// headers/body, status/size/latency) is logged and tagged with which
	// protocol served the request.
	mux.Handle("/mcp", logHTTP("streamable-http", server.NewStreamableHTTPServer(mcp, httpOpts...)))

	// The SSE server routes both /sse and /message through its ServeHTTP,
	// which also applies DNS-rebinding protection (the standalone SSEHandler
	// and MessageHandler do not).
	sse := server.NewSSEServer(mcp, sseOpts...)
	mux.Handle("/sse", logHTTP("sse", sse))
	mux.Handle("/message", logHTTP("sse", sse))

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}

// Run serves MCP over both transports — Streamable HTTP at /mcp and HTTP+SSE
// at /sse and /message — plus a /healthz liveness endpoint.
func Run(ctx context.Context, cfg *config.Config, svc *netboxx.Service) error {
	mux := newMux(cfg, svc)

	httpSrv := &http.Server{Addr: cfg.Listen, Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	log.Info().Str("listen", cfg.Listen).
		Str("streamable_http", "/mcp").Str("sse", "/sse").Str("sse_message", "/message").
		Bool("token_passthrough", cfg.TokenPassthrough).
		Msg("[mcpserver] starting")
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
