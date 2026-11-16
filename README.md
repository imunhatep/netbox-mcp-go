# netbox-mcp

MCP (Model Context Protocol) server exposing a NetBox instance to Claude Code
and other MCP clients. Read-only: it lists and fetches the organization
hierarchy (regions, site groups, sites, locations, tenant groups, tenants,
contacts), racks, devices, interfaces, prefixes (subnets), IP addresses, VLANs,
VRFs, clusters and virtual machines, plus the changelog (audit trail) — it never
writes to NetBox.

Objects are passed through exactly as NetBox returns them — including
`custom_fields`, `tags` and `comments` — so nothing is lost and any NetBox
version works. Only hyperlink noise (`url`/`display_url` keys) is stripped.
Every list tool accepts `brief: true` to get NetBox's compact representation
when full objects would be too verbose.

## Configuration

| Env var | Flag | Default | Description |
|---|---|---|---|
| `NETBOX_URL` | `--netbox-url` | — (required) | Base URL of the NetBox instance |
| `NETBOX_TOKEN` | `--netbox-token` | — (required) | NetBox API token; optional with token passthrough, where it is the fallback |
| `NETBOX_TOKEN_PASSTHROUGH` | `--token-passthrough` | `false` | Authenticate to NetBox with each request's own token (see below) |
| `NETBOX_MCP_LISTEN` | `--listen` | `127.0.0.1:3031` | HTTP listen address |
| `NETBOX_MCP_REQUEST_TIMEOUT` | `--request-timeout` | `30s` | Per-request timeout to NetBox |
| `NETBOX_MCP_PAGE_SIZE` | `--page-size` | `50` | Objects per page returned by list tools |
| `NETBOX_MCP_VERBOSE` | `--verbose` / `-v` | `3` | Log level: 0=fatal … 5=trace |
| `NETBOX_MCP_LOG_FORMAT` | `--log-format` | `console` | `console` (human-readable) or `json` (one object per line, for log collectors and `jq`) |

## Run

```sh
make build
NETBOX_URL=https://netbox.internal NETBOX_TOKEN=xxxx ./bin/netbox-mcp serve
```

Endpoints:

- `POST /mcp` — Streamable HTTP (current MCP spec)
- `GET /sse` + `POST /message` — legacy HTTP+SSE transport
- `GET /healthz` — liveness probe

### Token passthrough

By default every request is served with the single configured `NETBOX_TOKEN`, so
all callers share one service account's view of NetBox. With
`NETBOX_TOKEN_PASSTHROUGH=true` each MCP request instead authenticates with the
token from its own `Authorization: Bearer <token>` header, so callers see
exactly what their own NetBox permissions allow — the right default for a shared
or multi-tenant deployment.

```sh
NETBOX_URL=https://netbox.internal NETBOX_TOKEN_PASSTHROUGH=true ./bin/netbox-mcp serve
```

Details:

- The token is scoped to one HTTP request; it is never stored on the shared
  client. Tokens are redacted from trace logs.
- A token starting with `nbt_` (NetBox Labs Cloud) is sent to NetBox with the
  `Bearer` scheme, anything else with NetBox's `Token` scheme.
- Only the `Bearer` scheme is read from the incoming header; other schemes are
  ignored rather than treated as credentials.
- Requests without a bearer token fall back to `NETBOX_TOKEN`. If that is also
  unset, the tool call fails with an `unauthorized` message instead of querying
  NetBox anonymously — so leaving `NETBOX_TOKEN` unset makes authentication
  mandatory.
- Serve over TLS (or bind to loopback behind a proxy that terminates it) since
  callers now send their own credentials.

### Container

Published image (`NETBOX_URL`/`NETBOX_TOKEN` are passed through from the
current shell environment):

```sh
podman run --rm -d \
  --name netbox-mcp \
  -e NETBOX_URL \
  -e NETBOX_TOKEN \
  -p 127.0.0.1:3031:3031 ghcr.io/imunhatep/netbox-mcp-go:latest
```

Local build:

```sh
make image
podman run --rm -p 3031:3031 \
  -e NETBOX_URL=https://netbox.internal -e NETBOX_TOKEN=xxxx \
  netbox-mcp:dev
```

### Claude Code

```sh
claude mcp add --transport http netbox http://127.0.0.1:3031/mcp
```

### Agent skill

The binary embeds a Claude Code skill, `using-netbox-mcp`
([skills/using-netbox-mcp/SKILL.md](skills/using-netbox-mcp/SKILL.md)), that
teaches the model which tool to pick, how to batch lookups and how to keep
results small. It installs and updates itself:

- On connect the server's instructions announce the current skill version
  (server version + hash of the skill text). If
  `~/.claude/skills/using-netbox-mcp/SKILL.md` is missing or carries another
  `netbox-skill-version`, the model calls `get_skill` and writes the returned
  content to that path. Every new release therefore ships its matching skill.
- Claude Code asks before writing the file unless that path is allowed, and
  loads a newly written skill from the next session on.

Manual install or check:

```sh
netbox-mcp skill --install          # write it if missing or outdated
netbox-mcp skill --check            # compare installed vs current
mkdir -p ~/.claude/skills/using-netbox-mcp
podman run --rm ghcr.io/imunhatep/netbox-mcp-go:latest skill \
  > ~/.claude/skills/using-netbox-mcp/SKILL.md
```

## Tools

All list tools accept an optional 1-based `page` plus `brief` and return
`{count, page, page_size, has_next, results}`. Filters that reference another
object (`region`, `group`, `site`, `location`, `parent`, `rack`, `role`,
`cluster`, `type`, `tenant`, `device`, `vm`) accept either a numeric id or the
slug/name. `vrf` takes the id or the route distinguisher — NetBox does not
filter by VRF name.

`list_devices`, `list_virtual_machines`, `list_ip_addresses`, `list_prefixes`
and `list_clusters` also take `fields` (top-level field names, NetBox 4.0+),
which returns only those attributes and overrides `brief`. Array arguments
(`fields`, `names`, `ids`, `addresses`, `types`) also accept a comma-separated
string.

### Search

| Tool | Description |
|---|---|
| `search` | Global free-text search fanned out concurrently across regions, site groups, sites, locations, racks, devices, prefixes, IPs, VLANs, VRFs, clusters, VMs, tenant groups, tenants and contacts; brief results grouped by object type with per-type match counts; `limit` per type (default 10), `types` to narrow. An IP or CIDR query only searches prefixes and IP addresses. A type that fails is reported in its group's `error` |

### Organization (DCIM)

Regions, site groups and locations all nest; `parent` narrows a listing to one
node's direct children, and omitting it returns the flat list.

| Tool | Description |
|---|---|
| `list_regions` | List regions (geographic hierarchy); `search`, `parent` |
| `get_region` | One region by id or slug |
| `list_site_groups` | List site groups (functional grouping); `search`, `parent` |
| `get_site_group` | One site group by id or slug |
| `list_sites` | List sites; `search`, `region`, `group`, `tenant`, `status`, `tag` |
| `get_site` | One site by id or slug |
| `list_locations` | List locations within sites (buildings, floors, rooms, cages); `search`, `site`, `parent` |
| `get_location` | One location by id or slug |

### DCIM

| Tool | Description |
|---|---|
| `list_racks` | List racks; `search`, `site`, `location` |
| `list_devices` | List devices; `names`/`ids` (batch: many devices in one call), `fields`, `search`, `region`, `site`, `location`, `rack`, `role`, `tenant`, `status`, `tag` |
| `get_device` | One device by id or exact name |
| `list_device_interfaces` | Interfaces of a device |

### IPAM

| Tool | Description |
|---|---|
| `list_prefixes` | List subnets; `search`, `within` (parent CIDR), `contains` (IP/CIDR), `site`, `vrf`, `status`, `family`, `tag`, `fields` |
| `get_prefix` | One prefix by id or exact CIDR |
| `list_ip_addresses` | List IPs; `search`, `parent` (prefix), `device`, `vm`, `vrf`, `status`, `dns_name`, `tag`, `fields` |
| `lookup_ip_addresses` | Up to 100 addresses in one call: each address's IP objects, or for an unregistered address the prefixes that contain it |
| `get_ip_address` | One IP by id or address |
| `list_vlans` | List VLANs; `search`, `site`, `vid` |
| `list_vrfs` | List VRFs; `search` |

### Virtualization

| Tool | Description |
|---|---|
| `list_clusters` | List clusters; `search`, `type`, `group`, `site`, `tenant`, `status`, `tag`, `fields` |
| `list_virtual_machines` | List VMs; `names`/`ids` (batch), `fields`, `search`, `cluster`, `site`, `tenant`, `status`, `tag` |
| `get_virtual_machine` | One VM by id or exact name |
| `list_vm_interfaces` | Interfaces of a VM |

### Tenancy / misc

| Tool | Description |
|---|---|
| `list_tenant_groups` | List tenant groups; `search`, `parent` |
| `list_tenants` | List tenants; `search`, `group` |
| `get_tenant` | One tenant by id or slug |
| `list_contacts` | List contacts (people/teams assignable to objects); `search`, `group` |
| `netbox_status` | NetBox version and installed plugins |
| `get_skill` | The embedded `using-netbox-mcp` skill, matched to this server version (no NetBox call) |

### Changelog

| Tool | Description |
|---|---|
| `list_changelog` | Audit trail of created/updated/deleted objects, newest first, with `prechange_data`/`postchange_data` snapshots; `search`, `user`, `action`, `object_type` (app.model, e.g. `dcim.device`), `object_id`, `request_id`, `after`, `before` |

## Development

```sh
make test   # go test ./...
make build  # bin/netbox-mcp
make run    # build + serve
```

Layout mirrors [gitlab-mcp](https://github.com/imunhatep/gitlab-mcp):

- `internal/config` — flags/env validation
- `internal/netboxx` — lightweight NetBox REST client (full-object passthrough)
- `internal/mcpserver` — MCP server, transports, request logging
- `internal/mcpserver/tools` — tool definitions per NetBox domain
- `internal/skill` + `skills/using-netbox-mcp` — embedded agent skill, version marker, installer
