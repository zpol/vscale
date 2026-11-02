# User guide

## Install the agent

Download the binary for your OS from releases (or build):

```bash
# from source
./scripts/build-agent.sh
# Linux example
sudo install -m 755 dist/vscale_linux_amd64 /usr/local/bin/vscale
```

Requirements for bringing the tunnel up:

- Linux: `wireguard` kernel module + `wg-quick` (`wireguard-tools`)
- Windows/macOS: install [WireGuard](https://www.wireguard.com/install/); if `wg-quick` is missing, `vscale up` writes a `.conf` you can import into the WireGuard app

## Install (easiest)

Debian/Ubuntu:

```bash
curl -fsSL https://raw.githubusercontent.com/zpol/vscale/main/scripts/install.sh | sudo bash
# or: sudo apt install ./vscale_*_amd64.deb
```

## Join a vscale

Ask an admin for a **join token**, then:

```bash
vscale setup
```

Or manually:

```bash
vscale join <TOKEN> --server https://vscale.0ffsec.com --name laptop
vscale up
vscale status
```

State is stored in `~/.config/vscale/state.json` (override with `VSCALE_STATE`).

## Everyday commands

| Command | Purpose |
|---------|---------|
| `vscale status` | JSON status: address, peers, routes, local CIDRs |
| `vscale routes` | List AllowedIPs / routes from control plane |
| `vscale up` | Fetch config and bring tunnel up |
| `vscale down` | Tear tunnel down |
| `vscale daemon` | Keep syncing config + heartbeats (recommended) |

## Accessing services

Once up, you can reach:

- Overlay IPs of other devices/hubs in your vscale (`100.x.y.z`)
- Approved LAN routes advertised by hubs (ask your admin for the CIDR)

Example:

```bash
ping 100.64.0.1          # hub
curl http://10.50.0.10/  # if 10.50.0.0/24 is an approved route via hub
```

## Troubleshooting

- `not enrolled` → run `join` again
- `wg-quick not found` → install WireGuard tools or import the printed `.conf`
- Offline in admin UI → ensure `vscale daemon` is running and can reach the API
- No connectivity to LAN services → confirm route is **approved** and hub has IP forwarding
