package mcpserver

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imunhatep/netbox-mcp-go/internal/config"
	"github.com/imunhatep/netbox-mcp-go/internal/netboxx"
)

// startMCP boots the full MCP HTTP handler behind an httptest server pointed
// at the given NetBox base URL, performs the JSON-RPC initialize handshake and
// returns the server URL, the session ID and a ready HTTP client — exactly the
// flow a real MCP client (e.g. Claude Code) goes through when it connects.
func startMCP(t *testing.T, netboxURL string) (string, string, *http.Client) {
	t.Helper()
	cfg := &config.Config{
		NetBoxURL:      netboxURL,
		NetBoxToken:    "tok",
		RequestTimeout: 5 * time.Second,
		PageSize:       50,
	}
	svc, err := netboxx.NewService(cfg)
	require.NoError(t, err)

	ts := httptest.NewServer(server.NewStreamableHTTPServer(New(cfg, svc)))
	t.Cleanup(ts.Close)

	client := &http.Client{Timeout: 10 * time.Second}

	initBody := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{` +
		`"protocolVersion":"2024-11-05","capabilities":{},` +
		`"clientInfo":{"name":"integration-test","version":"0.0.1"}}}`
	resp := post(t, client, ts.URL, "", initBody)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	sessionID := resp.Header.Get("Mcp-Session-Id")
	require.NotEmpty(t, sessionID, "server must return a session ID from initialize")
	return ts.URL, sessionID, client
}

func post(t *testing.T, client *http.Client, url, sessionID, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

// TestToolsListOverHTTP asserts every tool group is exposed over the wire.
func TestToolsListOverHTTP(t *testing.T) {
	url, sid, client := startMCP(t, "http://netbox.invalid")

	resp := post(t, client, url, sid, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	out := buf.String()

	require.Equal(t, http.StatusOK, resp.StatusCode, "response body: %s", out)

	for _, tool := range []string{
		"search",
		"list_regions", "get_region", "list_site_groups", "get_site_group",
		"list_sites", "get_site", "list_locations", "get_location", "list_racks",
		"list_devices", "get_device", "list_device_interfaces",
		"list_prefixes", "get_prefix", "list_ip_addresses", "get_ip_address",
		"list_vlans", "list_vrfs",
		"list_clusters", "list_virtual_machines", "get_virtual_machine", "list_vm_interfaces",
		"list_tenant_groups", "list_tenants", "get_tenant", "list_contacts", "netbox_status",
		"list_changelog",
	} {
		assert.Contains(t, out, `"`+tool+`"`)
	}
}

// TestToolCallOverHTTP drives a full tools/call round trip: MCP client ->
// MCP server -> stubbed NetBox API -> slim DTO JSON back to the client.
func TestToolCallOverHTTP(t *testing.T) {
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/sites/", r.URL.Path)
		assert.Equal(t, "Token tok", r.Header.Get("Authorization"))
		assert.Equal(t, "ams", r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"next":null,"results":[{"id":1,"name":"AMS DC1",` +
			`"slug":"ams-dc1","status":{"value":"active","label":"Active"},` +
			`"device_count":12,"url":"http://ignored.example/api/dcim/sites/1/",` +
			`"custom_fields":{"cmdb_id":"X-100"},"tags":[{"id":9,"name":"prod"}],` +
			`"comments":"decom planned"}]}`))
	}))
	defer netbox.Close()

	url, sid, client := startMCP(t, netbox.URL)

	resp := post(t, client, url, sid,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_sites","arguments":{"search":"ams"}}}`)
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	out := buf.String()

	require.Equal(t, http.StatusOK, resp.StatusCode, "response body: %s", out)
	assert.Contains(t, out, "AMS DC1")
	assert.Contains(t, out, "ams-dc1")
	assert.Contains(t, out, `\"count\": 1`)
	// Full passthrough: custom_fields, tags and comments reach the model...
	assert.Contains(t, out, "custom_fields")
	assert.Contains(t, out, "cmdb_id")
	assert.Contains(t, out, "decom planned")
	assert.Contains(t, out, `\"prod\"`)
	// ...while hyperlink noise is stripped.
	assert.NotContains(t, out, "ignored.example")
}

// TestOrganizationToolCallOverHTTP drives an organization-hierarchy tool end to
// end and asserts the site/parent filters reach NetBox in id-or-slug form.
func TestOrganizationToolCallOverHTTP(t *testing.T) {
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/dcim/locations/", r.URL.Path)
		assert.Equal(t, "ams-dc1", r.URL.Query().Get("site"))
		assert.Equal(t, "4", r.URL.Query().Get("parent_id"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":1,"next":null,"results":[{"id":11,"name":"Cage 4",` +
			`"slug":"cage-4","site":{"id":1,"name":"AMS DC1","slug":"ams-dc1"},` +
			`"parent":{"id":4,"name":"Floor 2"},"rack_count":6,` +
			`"url":"http://ignored.example/api/dcim/locations/11/"}]}`))
	}))
	defer netbox.Close()

	url, sid, client := startMCP(t, netbox.URL)

	resp := post(t, client, url, sid,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_locations",`+
			`"arguments":{"site":"ams-dc1","parent":"4"}}}`)
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	out := buf.String()

	require.Equal(t, http.StatusOK, resp.StatusCode, "response body: %s", out)
	assert.Contains(t, out, "Cage 4")
	assert.Contains(t, out, "Floor 2")
	assert.Contains(t, out, `\"rack_count\": 6`)
	assert.NotContains(t, out, "ignored.example")
}

// TestSearchToolOverHTTP drives the global search tool end to end: the stub
// answers every fanned-out endpoint, with matches only on devices.
func TestSearchToolOverHTTP(t *testing.T) {
	netbox := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "core", r.URL.Query().Get("q"))
		assert.Equal(t, "true", r.URL.Query().Get("brief"))
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/dcim/devices/" {
			_, _ = w.Write([]byte(`{"count":2,"next":null,"results":[{"id":42,"name":"sw-core-01"},{"id":43,"name":"sw-core-02"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"count":0,"next":null,"results":[]}`))
	}))
	defer netbox.Close()

	url, sid, client := startMCP(t, netbox.URL)

	resp := post(t, client, url, sid,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"search","arguments":{"query":"core"}}}`)
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	out := buf.String()

	require.Equal(t, http.StatusOK, resp.StatusCode, "response body: %s", out)
	assert.Contains(t, out, "sw-core-01")
	assert.Contains(t, out, "sw-core-02")
	assert.Contains(t, out, `\"object_type\": \"devices\"`)
	// Empty groups are omitted from the result.
	assert.NotContains(t, out, `\"object_type\": \"tenants\"`)
}
