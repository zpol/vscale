# vscale

Self-hosted **hub-and-spoke VPN** (WireGuard) for home labs and small teams.

Not Tailscale — your own control plane, overlay IPs, hubs, and devices.

## Install the agent (home user)

### One-liner (Debian/Ubuntu)

```bash
curl -fsSL https://raw.githubusercontent.com/zpol/vscale/main/scripts/install.sh | sudo bash
```

### From a `.deb` release

```bash
sudo apt install ./vscale_*_amd64.deb
# needs: wireguard-tools (pulled in as dependency)
```

### Then connect

Ask your admin for a **join token**, then:

```bash
vscale setup
```

That asks for the server URL + token, enrolls this PC, and can bring the tunnel up.

Manual equivalent:

```bash
vscale join <TOKEN> --server https://vscale.0ffsec.com
vscale up
vscale status
```

### Everyday commands

| Command | Purpose |
|---------|---------|
| `vscale setup` | Interactive enroll + optional `up` |
| `vscale up` / `down` | Tunnel on/off |
| `vscale status` | Overlay IP, peers, routes |
| `vscale daemon` | Keep syncing (stay “online” in the UI) |

## Docs

- [Architecture](docs/architecture.md)
- [Networking](docs/networking.md)
- [User guide](docs/user-guide.md)
- [Admin guide](docs/admin-guide.md)
- [API](docs/api.md)

## Releases

GitHub Releases ship:

- `.deb` / `.rpm` agent packages (`amd64`, `arm64`)
- Agent archives for Linux / macOS / Windows
- `vscale-control-plane` Linux binaries

Tag a version to publish:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Local package without CI:

```bash
./scripts/build-deb.sh
# → dist/vscale_0.1.0_amd64.deb
```

## Admin / control plane

```bash
cd deploy && docker compose up -d --build
```

- Web http://localhost:3000 — API http://localhost:8080  
- Default login `admin` / `admin123` (override with env)

Create a **vscale** → **Add hub** on a VPS → issue a **join token** → users run `vscale setup`.

## Repository layout

```
cmd/control-plane   # API server
cmd/vscale          # agent (client + hub)
internal/           # models, auth, allocator, api, agent…
web/                # admin UI (React)
deploy/             # docker compose + k8s + Dockerfiles
packaging/          # deb/rpm metadata
docs/
scripts/            # build-deb, install, e2e…
```

## License

MIT. WireGuard is a trademark of Jason A. Donenfeld.
