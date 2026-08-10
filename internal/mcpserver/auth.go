package mcpserver

import (
	"context"
	"net/http"
	"strings"

	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"

	"github.com/rs/zerolog/log"
)

// bearerToken extracts a token from an Authorization: Bearer header. Only the
// Bearer scheme is recognised; anything else (Basic, or NetBox's own Token
// scheme) yields "" so a stray header cannot be misread as a credential.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// tokenPassthroughContext lifts each request's bearer token into the context
// the tool handlers run under, so netboxx authenticates to NetBox as the
// caller. Requests without a usable header fall through to the server's
// configured token.
//
// This is the transport-level half of passthrough auth: the token is scoped to
// one HTTP request and never stored on the shared *netboxx.Service.
func tokenPassthroughContext(ctx context.Context, r *http.Request) context.Context {
	token := bearerToken(r)
	if token == "" {
		log.Debug().Str("path", r.URL.Path).
			Msg("[mcpserver] no bearer token on request, using configured NetBox token")
		return ctx
	}
	return netboxx.WithToken(ctx, token)
}
