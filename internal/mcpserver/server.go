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
	"github.com/imunhatep/netbox-mcp-go/internal/version"
)

// New builds the MCP server and registers every tool group. All tools are
// read-only: the server never writes to NetBox.
func New(cfg *config.Config, svc *netboxx.Service) *server.MCPServer {
	s := server.NewMCPServer(
		"netbox-mcp",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
		server.WithHooks((&mcpLogger{}).hooks()),
	)
	tools.RegisterSearch(s, svc)
	tools.RegisterDCIM(s, svc)
	tools.RegisterIPAM(s, svc)
	tools.RegisterVirtualization(s, svc)
	tools.RegisterTenancy(s, svc)
	return s
}

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

	mux := http.NewServeMux()
	// Each transport is wrapped so the HTTP layer (client IP, method, path,
	// headers/body, status/size/latency) is logged and tagged with which
	// protocol served the request.
	mux.Handle("/mcp", logHTTP("streamable-http", server.NewStreamableHTTPServer(mcp)))

	// The SSE server routes both /sse and /message through its ServeHTTP,
	// which also applies DNS-rebinding protection (the standalone SSEHandler
	// and MessageHandler do not).
	sse := server.NewSSEServer(mcp)
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
		Msg("[mcpserver] starting")
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
