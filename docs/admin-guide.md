# Admin guide

## Quick start (docker compose)

```bash
cd deploy
export PUBLIC_URL=http://localhost:8080
docker compose up -d --build
```

- API: http://localhost:8080  
- Web: http://localhost:3000  
- Login: `admin` / `admin123` (set `DEFAULT_ADMIN_PASSWORD`)

## Quick start (k3s lab @ 192.168.68.113)

```bash
./scripts/deploy-k3s.sh
```

- API NodePort: http://192.168.68.113:30808  
- Web NodePort: http://192.168.68.113:30300  

## Create a vscale

UI: **Vscales → Create**  
API:

```bash
curl -X POST "$API/api/vscales" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"lab","description":"home lab"}'
```

The allocator assigns the next free overlay `/16` inside `100.64.0.0/10`.

## Add a hub (VPS)

1. Create hub in UI (provide public endpoint, e.g. VPS IP).
2. Copy the **agent token**.
3. On the VPS:

```bash
vscale hub --server http://192.168.68.113:30808 --token <HUB_TOKEN>
# or: save token then `vscale daemon`
```

Open UDP **51820** (or your listen port) toward the hub.

## Issue join tokens

UI: Vscales → join token icon  
Or API `POST /api/vscales/:id/join-tokens` with `reusable`, `uses`, `expiresInHours`.

Distribute tokens out-of-band; revoke when leaked.

## Advertise a LAN / service subnet

```bash
curl -X POST "$API/api/vscales/$ID/routes" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"cidr":"10.50.0.0/24","viaType":"hub","viaId":1,"autoApprove":true}'
```

If the allocator reports conflicts, fix the CIDR or use `"force": true` only when intentional.

On the hub, ensure forwarding/NAT as needed for the LAN behind it.

## Users & RBAC

Global roles:

| Role | Powers |
|------|--------|
| `admin` | Full control, user CRUD |
| `operator` | Create/manage vscales |
| `member` | Access only vscales they belong to |

Membership roles per vscale: `owner`, `admin`, `member`.

## Topology & stats

- Dashboard shows vscale/hub/device counts and online (heartbeat &lt; 2m)
- Topology page draws hub↔device spokes with overlay addresses and CIDR labels
- `GET /api/vscales/:id/topology` for automation

## Revoke a device

UI Devices → delete, or `DELETE /api/vscales/:id/devices/:deviceId`.  
IP is released for reuse; hub peer list updates on next sync.

## E2E smoke test

```bash
./scripts/build-agent.sh
VSCALE_API=http://192.168.68.113:30808 ./scripts/e2e-api.sh
```
