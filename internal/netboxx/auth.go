package netboxx

import (
	"context"
	"fmt"
	"strings"
)

// tokenCtxKey carries a per-request NetBox token through the context. Its type
// is unexported so no other package can collide with or forge the key.
type tokenCtxKey struct{}

// WithToken returns a context carrying a NetBox API token for this request.
// Every Service call made with that context authenticates as the token's user
// instead of the server's configured token, so a multi-user deployment can let
// each caller bring their own credentials — and inherit their NetBox object
// permissions — rather than sharing one service account.
//
// An empty token is ignored so callers can pass whatever they extracted
// without branching.
func WithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenCtxKey{}, token)
}

// tokenFromContext returns the per-request token, if any.
func tokenFromContext(ctx context.Context) string {
	t, _ := ctx.Value(tokenCtxKey{}).(string)
	return t
}

// nbtPrefix marks a NetBox Labs Cloud token, which authenticates with the
// Bearer scheme; a self-hosted NetBox API key uses the Token scheme.
const nbtPrefix = "nbt_"

// authHeader builds the Authorization value for a token, picking the scheme
// from the token's shape.
func authHeader(token string) string {
	if strings.HasPrefix(token, nbtPrefix) {
		return "Bearer " + token
	}
	return "Token " + token
}

// errNoToken is returned when neither the request nor the server configuration
// supplied a credential. It only happens with token passthrough enabled and no
// fallback token configured.
var errNoToken = fmt.Errorf("%w: no NetBox token — pass one as an Authorization: Bearer header, "+
	"or configure NETBOX_TOKEN as a fallback", ErrUnauthorized)

// resolveToken picks the per-request token when present, else the configured
// one.
func (s *Service) resolveToken(ctx context.Context) (string, error) {
	if t := tokenFromContext(ctx); t != "" {
		return t, nil
	}
	if s.token == "" {
		return "", errNoToken
	}
	return s.token, nil
}
