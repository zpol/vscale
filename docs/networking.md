# Networking

## Overlay pool

Global pool (default): **`10.66.0.0/16`** (RFC1918), with **`/24` per vscale**.

We deliberately avoid **`100.64.0.0/10`** (CGNAT). That range is used by **Tailscale**; installing `AllowedIPs = 100.64.0.0/16` on a client steals routes from Tailscale and often looks like “I lost the internet” (DNS/MagicDNS break).

Examples:

| Vscale # | Overlay |
|----------|---------|
| 1 | `10.66.0.0/24` |
| 2 | `10.66.1.0/24` |
| … | … |

Override with `VSCALE_GLOBAL_POOL` / `VSCALE_PREFIX_LEN` if needed — keep clear of Tailscale and your LAN.

Inside a vscale:

- Hub gets the first usable host `/32` (typically `.1`)
- Devices get subsequent `/32`s with hole reuse when revoked

## Hub-and-spoke paths

```
device A ──WG──► hub ──► advertised LAN/service routes
device B ──WG──► hub ──► overlay peers (via hub)
```

Client `AllowedIPs` include the vscale overlay CIDR plus **approved** routes. That lets devices reach:

1. Other overlay IPs (hairpinned via hub)
2. Services behind a hub (or gateway device) subnet route

## Conflict detection

The allocator rejects (severity `error`) when a candidate CIDR overlaps:

1. Any recorded `IpAllocation` in scope (overlay blocks, host `/32`s, approved routes)
2. **Local CIDRs** reported by the agent (interfaces/routes on the client machine)

RFC1918 ranges (`10/8`, `172.16/12`, `192.168/16`) produce **warnings** when used as advertised routes (intentional LAN exposure). Admins can `force` approve when they knowingly route a LAN.

## Agent local intelligence

On `up` / heartbeat the agent sends `localCidrs`. The control plane returns warnings if the overlay would collide with the client’s LAN — preventing “weird” routing where VPN IPs shadow home Wi‑Fi.

## Ports

| Path | Proto | Default |
|------|-------|---------|
| Control plane API | TCP | 8080 (NodePort 30808 in lab) |
| Admin web | TCP | 3000 (NodePort 30300) |
| Hub WireGuard | UDP | 51820 (per hub) |

## MTU / DNS

- Default MTU `1420`
- Client WireGuard configs do **not** set `DNS =` (avoids breaking local/Tailscale DNS). Overlay DNS metadata on a vscale is optional/admin-only.
