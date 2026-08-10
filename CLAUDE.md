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
`virtualization.go`, `tenancy.go`, `core.go`), and an `add(mcp.NewTool(...), handler)`
block in the mirror file under [tools/](internal/mcpserver/tools/). Also extend
the domain's backend interface (e.g. `tools.DCIMBackend` in
[tools/dcim.go](internal/mcpserver/tools/dcim.go)) — the tool packages depend on
these narrow interfaces, not on `*netboxx.Service`, so tests can substitute fakes.

### netboxx conventions

- **Full-object passthrough.** There are no DTOs: `netboxx.Object` is
  `map[string]any` and NetBox JSON is returned verbatim so `custom_fields`,
  `tags` and `comments` survive and any NetBox version works. Do not introduce
  typed structs for NetBox objects. Nothing in the codebase reads a field out of
  a response — objects go decode → `prune()` → JSON text — so response structs
  would add no type safety while silently dropping per-deployment
  `custom_fields` and version-specific keys on re-marshal.
- **Requests, by contrast, are always typed.** Every `List*` method has the
  signature `List<Thing>(ctx, f <Thing>Filter, page int)` — one filter struct per
  endpoint, even single-field ones (`VRFFilter`, `VMInterfaceFilter`). No
  positional string parameters: adjacent same-typed args are silently swappable,
  and a new filter otherwise churns the signature and both backend interfaces.
- `prune()` recursively deletes only `url`/`display_url` keys. It is the sole
  sanctioned mutation of a response — call it on every object returned from a
  new endpoint.
- All requests go through `Service.get()`, which prefixes `/api/`, ensures a
  trailing slash, and sets the auth header. `list()` converts the
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
- **Auth is per request, not per Service.** [auth.go](internal/netboxx/auth.go)
  resolves the credential for every call: a token put on the context by
  `WithToken` wins, `Service.token` is the fallback, and no token at all is an
  `ErrUnauthorized` before any HTTP call. `authHeader` picks the scheme from the
  token — `Bearer` for `nbt_`-prefixed NetBox Labs Cloud tokens, `Token`
  otherwise. Never read `s.token` directly in a request path.
- Endpoints that moved between NetBox versions are tried newest-first with a
  404 fallback (see `changelogPaths` in [core.go](internal/netboxx/core.go),
  where ObjectChange moved from `extras` to `core` in NetBox 4.0). Only
  `ErrNotFound` falls through; any other error is returned as-is.
- [search.go](internal/netboxx/search.go) fans out concurrently over
  `searchTargets` because NetBox's REST API has no cross-type search endpoint.
  New object types should be added to that slice as well as getting their own
  list tool.

### mcpserver

One `*server.MCPServer` backs both transports: Streamable HTTP at `/mcp` and
legacy HTTP+SSE at `/sse` + `/message`, plus `/healthz`. Both are wrapped in
`logHTTP`, and MCP-level tool calls are logged via `mcpLogger.hooks()`.

With `--token-passthrough`, both transports get a context func
([auth.go](internal/mcpserver/auth.go)) that lifts each request's
`Authorization: Bearer` token into the handler context for `netboxx.WithToken`,
so callers query NetBox with their own permissions. It is off by default and
`Authorization` is redacted in both logging layers.

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
- [internal/mcpserver/auth_test.go](internal/mcpserver/auth_test.go) — token
  passthrough end to end through `newMux`: which Authorization header NetBox
  ends up seeing for a caller token, a cloud token, no header, and with
  passthrough off.

Uses `testify` `require`/`assert`.
