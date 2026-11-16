package mcpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rpcMethod names the JSON-RPC method for the access log and leaves the body
// intact for the MCP handler.
func TestRPCMethod(t *testing.T) {
	cases := map[string]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`: "tools/call",
		`{"jsonrpc":"2.0","id":7,"result":{}}`:                       "",
		`[{"jsonrpc":"2.0","id":1,"method":"ping"}]`:                 "batch",
		`not json`: "",
	}
	for body, want := range cases {
		r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
		assert.Equal(t, want, rpcMethod(r), body)
		rest, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, string(rest), "body must be restored")
	}
	assert.Empty(t, rpcMethod(httptest.NewRequest(http.MethodGet, "/mcp", nil)))
}

func TestShortSession(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	assert.Equal(t, "none", shortSession(r))
	r.Header.Set("Mcp-Session-Id", "mcp-session-01cc6e74-5c09-42bb-8cf7-563cf1083e80")
	assert.Equal(t, "01cc6e74", shortSession(r))
	r = httptest.NewRequest(http.MethodPost, "/message?sessionId=abc", nil)
	assert.Equal(t, "abc", shortSession(r))
}

func TestArgsPreviewAndToolError(t *testing.T) {
	assert.Equal(t, `{"names":["a","b"]}`, argsPreview(map[string]any{"names": []string{"a", "b"}}))
	assert.Equal(t, "{}", argsPreview(nil))
	long := argsPreview(map[string]any{"q": strings.Repeat("x", 2*maxArgsLog)})
	assert.True(t, strings.HasSuffix(long, "...(truncated)"))

	res := mcp.NewToolResultError("not found: no dcim/devices with name=\"x\"")
	assert.Contains(t, toolErrorText(res), "not found")
}
