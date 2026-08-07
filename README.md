# netbox-mcp

MCP (Model Context Protocol) server exposing a NetBox instance to Claude Code
and other MCP clients. Read-only: it lists and fetches the organization
hierarchy (regions, site groups, sites, locations, tenant groups, tenants,
contacts), racks, devices, interfaces, prefixes (subnets), IP addresses, VLANs,
VRFs, clusters and virtual machines — it never writes to NetBox.

Objects are passed through exactly as NetBox returns them — including
`custom_fields`, `tags` and `comments` — so nothing is lost and any NetBox
version works. Only hyperlink noise (`url`/`display_url` keys) is stripped.
Every list tool accepts `brief: true` to get NetBox's compact representation
when full objects would be too verbose.

## Configuration

| Env var | Flag | Default | Description |
|---|---|---|---|
| `NETBOX_URL` | `--netbox-url` | — (required) | Base URL of the NetBox instance |
| `NETBOX_TOKEN` | `--netbox-token` | — (required) | NetBox API token |
| `NETBOX_MCP_LISTEN` | `--listen` | `127.0.0.1:3031` | HTTP listen address |
| `NETBOX_MCP_REQUEST_TIMEOUT` | `--request-timeout` | `30s` | Per-request timeout to NetBox |
| `NETBOX_MCP_PAGE_SIZE` | `--page-size` | `50` | Objects per page returned by list tools |
| `NETBOX_MCP_VERBOSE` | `--verbose` / `-v` | `3` | Log level: 0=fatal … 5=trace |

## Run

```sh
make build
NETBOX_URL=https://netbox.internal NETBOX_TOKEN=xxxx ./bin/netbox-mcp serve
```

Endpoints:

- `POST /mcp` — Streamable HTTP (current MCP spec)
- `GET /sse` + `POST /message` — legacy HTTP+SSE transport
- `GET /healthz` — liveness probe

### Container

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

## Tools

All list tools accept an optional 1-based `page` plus `brief` and return
`{count, page, page_size, has_next, results}`. Filters that reference another
object (`region`, `group`, `site`, `location`, `parent`, `rack`, `role`, `vrf`,
`cluster`, `tenant`, `device`, `vm`) accept either a numeric id or the
slug/name.

### Search

| Tool | Description |
|---|---|
| `search` | Global free-text search fanned out concurrently across regions, site groups, sites, locations, racks, devices, prefixes, IPs, VLANs, VRFs, clusters, VMs, tenant groups, tenants and contacts; brief results grouped by object type with per-type match counts; `limit` per type (default 10) |

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
| `list_devices` | List devices; `search`, `region`, `site`, `location`, `rack`, `role`, `tenant`, `status`, `tag` |
| `get_device` | One device by id or exact name |
| `list_device_interfaces` | Interfaces of a device |

### IPAM

| Tool | Description |
|---|---|
| `list_prefixes` | List subnets; `search`, `within` (parent CIDR), `site`, `vrf`, `status`, `family`, `tag` |
| `get_prefix` | One prefix by id or exact CIDR |
| `list_ip_addresses` | List IPs; `search`, `parent` (prefix), `device`, `vm`, `vrf`, `status`, `dns_name`, `tag` |
| `get_ip_address` | One IP by id or address |
| `list_vlans` | List VLANs; `search`, `site`, `vid` |
| `list_vrfs` | List VRFs; `search` |

### Virtualization

| Tool | Description |
|---|---|
| `list_clusters` | List clusters; `search`, `site` |
| `list_virtual_machines` | List VMs; `search`, `cluster`, `site`, `tenant`, `status`, `tag` |
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
