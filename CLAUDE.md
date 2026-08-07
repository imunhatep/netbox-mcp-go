# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
make test                      # go test ./...
make build                     # -> bin/netbox-mcp (injects version/commit via ldflags)
make run                       # build + ./bin/netbox-mcp serve
make image                     # podman build -t netbox-mcp:dev
make tidy

go test ./internal/netboxx/ -run TestGetOneByName -v        # single test
go test ./internal/mcpserver/ -run TestToolsListOverHTTP -v  # single integration test
```

Running the server requires `NETBOX_URL` and `NETBOX_TOKEN` (see README for the
full flag/env table). Default listen address is `127.0.0.1:3031`; `-v 5` gives
trace logging including redacted request headers.

## Architecture

Four layers, each depending only on the one below:

```
cmd/netbox-mcp/main.go    -> internal.NewApp() + command.{Serve,Version}Command
internal/command/serve.go -> config.FromCommand + Validate, then mcpserver.Run
internal/mcpserver        -> MCP server, two HTTP transports, request logging
internal/mcpserver/tools  -> one Register* per NetBox domain; arg parsing only
internal/netboxx          -> read-only NetBox REST client
```

**Adding a tool** touches exactly two files: a method on `*netboxx.Service` in
the matching domain file ([dcim.go](internal/netboxx/dcim.go), `ipam.go`,
`virtualization.go`, `tenancy.go`), and an `add(mcp.NewTool(...), handler)`
block in the mirror file under [tools/](internal/mcpserver/tools/). Also extend
the domain's backend interface (e.g. `tools.DCIMBackend` in
[tools/dcim.go](internal/mcpserver/tools/dcim.go)) — the tool packages depend on
these narrow interfaces, not on `*netboxx.Service`, so tests can substitute fakes.

### netboxx conventions

- **Full-object passthrough.** There are no DTOs: `netboxx.Object` is
  `map[string]any` and NetBox JSON is returned verbatim so `custom_fields`,
  `tags` and `comments` survive and any NetBox version works. Do not introduce
  typed structs for NetBox objects.
- `prune()` recursively deletes only `url`/`display_url` keys. It is the sole
  sanctioned mutation of a response — call it on every object returned from a
  new endpoint.
- All requests go through `Service.get()`, which prefixes `/api/`, ensures a
  trailing slash, and sets the `Token` auth header. `list()` converts the
  1-based `page` argument into NetBox `limit`/`offset` and wraps results in
  `Page{count, page, page_size, has_next, results}`.
- `getOne()` accepts a numeric id (direct `GET {path}/{id}/`) *or* a symbolic
  ref via a lookup param — `slug` for sites, `name` for devices/VMs, `prefix`
  for prefixes. An ambiguous lookup is an error, not a silent first-match.
- Query helpers enforce the id-or-slug contract: `setRef` emits `site_id=5` for
  numerics and `site=<slug>` otherwise; `setStr` skips empties; `setBrief` adds
  `brief=true`. Use them rather than touching `url.Values` directly.
- Errors map non-2xx responses onto the sentinels in
  [errors.go](internal/netboxx/errors.go) (`ErrNotFound`, `ErrUnauthorized`,
  `ErrForbidden`, `ErrValidation`); wrap with `%w` so `errors.Is` keeps working.
- [search.go](internal/netboxx/search.go) fans out concurrently over
  `searchTargets` because NetBox's REST API has no cross-type search endpoint.
  New object types should be added to that slice as well as getting their own
  list tool.

### mcpserver

One `*server.MCPServer` backs both transports: Streamable HTTP at `/mcp` and
legacy HTTP+SSE at `/sse` + `/message`, plus `/healthz`. Both are wrapped in
`logHTTP`, and MCP-level tool calls are logged via `mcpLogger.hooks()`.

Tool handlers never return a Go error to the MCP layer — failures come back as
`errResult(err)` (an `mcp.NewToolResultError`) so the model sees the message.
Results are `jsonResult(v)`, indented JSON as text.

**The server is read-only by design.** Every tool is a GET; do not add write
tools without an explicit request.

## Tests

- [internal/netboxx/client_test.go](internal/netboxx/client_test.go) — service
  methods against an `httptest` NetBox stub; assert on the outgoing path/query
  as well as the decoded result, and that pruning kept `custom_fields`/`tags`
  while dropping `url`.
- [internal/mcpserver/integration_test.go](internal/mcpserver/integration_test.go)
  — drives the real JSON-RPC handshake (`initialize` → `Mcp-Session-Id` →
  `tools/list` / `tools/call`) over `httptest`. New tools should show up here.

Uses `testify` `require`/`assert`.
