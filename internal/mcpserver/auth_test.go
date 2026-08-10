package mcpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

func TestBearerToken(t *testing.T) {
	for _, tc := range []struct{ name, header, want string }{
		{"bearer", "Bearer abc123", "abc123"},
		{"lowercase scheme", "bearer abc123", "abc123"},
		{"padded", "Bearer   abc123  ", "abc123"},
		{"netbox token scheme is not a bearer token", "Token abc123", ""},
		{"basic", "Basic dXNlcjpwdw==", ""},
		{"empty", "", ""},
		{"scheme only", "Bearer ", ""},
		{"shorter than scheme", "Bear", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tc.header != "" {
				r.Header.Set("Authorization", tc.header)
			}
			assert.Equal(t, tc.want, bearerToken(r))
		})
	}
}

// startPassthroughMCP boots the real mux with token passthrough enabled and
// completes the initialize handshake, returning the /mcp URL, session id and
// client.
func startPassthroughMCP(t *testing.T, netboxURL, fallbackToken string) (string, string, *http.Client) {
	t.Helper()
	cfg := &config.Config{
		NetBoxURL:        netboxURL,
		NetBoxToken:      fallbackToken,
		RequestTimeout:   5 * time.Second,
		PageSize:         50,
		TokenPassthrough: true,
	}
	svc, err := netboxx.NewService(cfg)
	require.NoError(t, err)

	ts := httptest.NewServer(newMux(cfg, svc))
	t.Cleanup(ts.Close)
	url := ts.URL + "/mcp"

	client := &http.Client{Timeout: 10 * time.Second}
	resp := post(t, client, url, "", `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{`+
		`"protocolVersion":"2024-11-05","capabilities":{},`+
		`"clientInfo":{"name":"auth-test","version":"0.0.1"}}}`)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	sessionID := resp.Header.Get("Mcp-Session-Id")
	require.NotEmpty(t, sessionID)
	return url, sessionID, client
}

// callSites issues a list_sites tool call, optionally with an Authorization
// header, and returns the Authorization header NetBox saw.
func callSites(t *testing.T, url, sessionID string, client *http.Client, authHeader string) string {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":` +
		`{"name":"list_sites","arguments":{}}}`
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sessionID)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "response body: %s", out)
	return string(out)
}

// TestTokenPassthroughOverHTTP asserts the caller's bearer token — not the
// server's configured one — authenticates the outgoing NetBox request, and
// that the scheme is chosen from the token's shape.
func TestTokenPassthroughOverHTTP(t *testing.T) {
	var gotAuth string
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	}))
	defer netbox.Close()

	url, sessionID, client := startPassthroughMCP(t, netbox.URL, "server-fallback")

	// A self-hosted NetBox API key keeps NetBox's own Token scheme...
	callSites(t, url, sessionID, client, "Bearer caller-key")
	assert.Equal(t, "Token caller-key", gotAuth)

	// ...while a NetBox Labs Cloud token (nbt_ prefix) uses Bearer.
	callSites(t, url, sessionID, client, "Bearer nbt_cloudkey")
	assert.Equal(t, "Bearer nbt_cloudkey", gotAuth)

	// No header at all falls back to the configured token.
	callSites(t, url, sessionID, client, "")
	assert.Equal(t, "Token server-fallback", gotAuth)

	// A non-bearer header is ignored rather than forwarded as a credential.
	callSites(t, url, sessionID, client, "Basic dXNlcjpwdw==")
	assert.Equal(t, "Token server-fallback", gotAuth)
}

// TestTokenPassthroughDisabled asserts the header is ignored entirely unless
// passthrough is switched on — the default must not forward caller credentials.
func TestTokenPassthroughDisabled(t *testing.T) {
	var gotAuth string
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	}))
	defer netbox.Close()

	cfg := &config.Config{
		NetBoxURL:      netbox.URL,
		NetBoxToken:    "tok",
		RequestTimeout: 5 * time.Second,
		PageSize:       50,
	}
	svc, err := netboxx.NewService(cfg)
	require.NoError(t, err)
	ts := httptest.NewServer(newMux(cfg, svc))
	defer ts.Close()

	client := &http.Client{Timeout: 10 * time.Second}
	resp := post(t, client, ts.URL+"/mcp", "", `{"jsonrpc":"2.0","id":0,"method":"initialize",`+
		`"params":{"protocolVersion":"2024-11-05","capabilities":{},`+
		`"clientInfo":{"name":"auth-test","version":"0.0.1"}}}`)
	sessionID := resp.Header.Get("Mcp-Session-Id")
	resp.Body.Close()
	require.NotEmpty(t, sessionID)

	callSites(t, ts.URL+"/mcp", sessionID, client, "Bearer caller-key")
	assert.Equal(t, "Token tok", gotAuth)
}

// TestNoTokenAtAll asserts a passthrough deployment with no fallback token
// rejects an unauthenticated call with a clear message instead of querying
// NetBox anonymously.
func TestNoTokenAtAll(t *testing.T) {
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("NetBox must not be called without a token")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer netbox.Close()

	url, sessionID, client := startPassthroughMCP(t, netbox.URL, "")
	out := callSites(t, url, sessionID, client, "")
	assert.Contains(t, out, "no NetBox token")
}
