---
name: using-netbox-mcp
description: Use when answering questions from NetBox through the netbox MCP server — what a device, VM, IP address, prefix, VLAN, VRF, site, rack, cluster or tenant is and where it lives, which network an IP belongs to, what is in a site/rack/cluster/tenant, who owns something, custom fields and tags, and who changed what and when (changelog). Covers tool choice, batching many hosts or IPs into one call, fields/brief/paging to keep results small, the id-or-slug and VRF (route distinguisher) rules, reading results, and the call patterns that waste calls.
---

<!-- netbox-skill-version: {{VERSION}} -->

# Using netbox-mcp

netbox-mcp is a **read-only** view of NetBox: every tool is a GET and nothing
can change NetBox through it. Objects come back as NetBox returns them, minus
hyperlink keys, so `custom_fields`, `tags` and `comments` are always there.

## Pick the tool

| Question | Tool |
|---|---|
| "What is X?" when the type of X is unknown (hostname fragment, tag value, serial) | `search` |
| What are these IPs / which network is this IP in | `lookup_ip_addresses` (many addresses, one call) |
| Details of several devices / VMs | `list_devices` / `list_virtual_machines` with `names=[...]` or `ids=[...]` |
| Details of one device / VM / IP / prefix | `get_device`, `get_virtual_machine`, `get_ip_address`, `get_prefix` |
| Details of one site / region / site group / location / tenant | `get_site`, `get_region`, `get_site_group`, `get_location`, `get_tenant` |
| Which prefixes contain an address | `list_prefixes contains=<ip>` |
| Subnets inside a block | `list_prefixes within=<cidr>` |
| IPs inside a subnet | `list_ip_addresses parent=<cidr>` |
| IPs of a device / VM | `list_ip_addresses device=<name>` / `vm=<name>` |
| Interfaces | `list_device_interfaces` / `list_vm_interfaces` |
| Everything at a site / rack / tenant / cluster | `list_devices` / `list_virtual_machines` with that filter |
| Org hierarchy | `list_regions`, `list_site_groups`, `list_sites`, `list_locations`, `list_racks`, `list_tenant_groups`, `list_tenants`, `list_contacts` |
| VLANs, VRFs, clusters | `list_vlans`, `list_vrfs`, `list_clusters` |
| Who changed what, when | `list_changelog` (for one object: `object_type=dcim.device object_id=42`) |
| NetBox version and plugins | `netbox_status` |
| This skill | `get_skill` |

## Batch, never loop

The most common waste is one call per object. Each call costs a round trip
and repeats the whole JSON envelope.

- **Many IPs:** `lookup_ip_addresses addresses=["10.8.3.66","10.8.3.67",...]`
  (up to 100). Each address comes back with `found`, its IP objects, or for an
  unregistered address its `containing_prefixes`. A miss already tells you the
  network, so **do not probe neighbouring addresses** one by one. To see what
  is registered in a range use `list_ip_addresses parent=<cidr>` once.
- **Many devices / VMs:** `list_devices names=["a","b","c"]` or `ids=[1,2,3]`
  returns full objects for all of them in one call. Do not follow a brief
  listing with `get_device` per row.
- **Many attributes across many objects:** add `fields=[...]`, e.g.
  `list_devices site=fra1 fields=["name","status","role","primary_ip4","custom_fields"]`.
  `id` is always included. Only top-level names work (`site`, not `site.name`);
  related objects come back in brief form. A misspelt field is silently dropped.
- **Unknown names:** `list_devices search=<fragment> brief=true` to discover
  names, then one batched call for details.

`search` with an IP or CIDR only queries prefixes and IP addresses. Its
`search` match on addresses is a text prefix match: `10.1.2.3` also matches
`10.1.2.30`–`10.1.2.39`. For exact addresses use `lookup_ip_addresses`.

## Keep results small

- `brief=true` gives id + name/display only: use it to discover, not to answer.
- `fields=[...]` gives exactly the attributes needed; it overrides `brief`.
- Filter before paging. `list_clusters` can have hundreds of rows: filter by
  `type`, `group`, `site`, `tenant`, `status` or `search`. Never page through a
  whole table of full objects to find one thing.
- `search limit` caps results per type (default 10); narrow with
  `types=["devices","virtual_machines"]`.

## Arguments

- **id or slug/name:** reference filters (`site`, `region`, `tenant`, `rack`,
  `role`, `cluster`, `type`, `group`, `location`) accept a numeric id or a
  slug/name. A number is always treated as an id.
- **VRF filters take the id or the route distinguisher**, not the VRF name.
  Resolve a name first with `list_vrfs search=<name>` and pass its `id`.
- `get_*` by name must match exactly one object. An ambiguous name is an
  error: use the numeric id (VM names can repeat across clusters).
- `get_ip_address` accepts `10.1.0.5` (any mask) or `10.1.0.5/24` (exact).
- Dates for `list_changelog after/before`: `2026-08-01` or RFC 3339.

## Reading results

- List tools return `{count, page, page_size, has_next, results}`. `count` is
  the total match count; when `has_next` is true, ask for `page=2` only if you
  really need more, otherwise narrow the filter.
- `lookup_ip_addresses` returns one entry per requested address, in order:
  `found`, `ip_addresses` (full objects: `assigned_object`, `dns_name`,
  `status`, `tenant`, `vrf`) or `containing_prefixes` (most general first; the
  last is the most specific). `error` on one entry does not affect the others.
- `search` returns only non-empty groups; a group with `error` failed on its
  own while the rest succeeded.
- A device's management address is `primary_ip4` / `primary_ip6` / `oob_ip`.
  `custom_fields` hold site-specific data (owners, support contracts, ...);
  read them before saying "unknown".
- Changelog records carry `prechange_data` and `postchange_data`; diff them to
  say what changed.

## Errors

- `not found`: the id/name does not exist (or your token cannot see it).
- `invalid argument`: bad filter value, ambiguous name, unknown search type or
  a nested field name — the message says which.
- `unauthorized` / `forbidden`: the server's NetBox token (or, with token
  passthrough, your own) lacks access. Do not retry the same call.

## Keeping this skill current

The server announces the current skill version on connect. If
`~/.claude/skills/using-netbox-mcp/SKILL.md` lacks the matching
`netbox-skill-version` line, call `get_skill` with the installed version and
write `content` verbatim to that path. Manual install:
`netbox-mcp skill --install`, or
`podman run --rm ghcr.io/imunhatep/netbox-mcp-go:latest skill > ~/.claude/skills/using-netbox-mcp/SKILL.md`.
