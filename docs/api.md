# API reference

Base URL examples: `http://localhost:8080` or `http://192.168.68.113:30808`.

Auth for user routes: `Authorization: Bearer <jwt>`  
Auth for agent routes: `X-Vscale-Token: <agentToken>`

## Health

- `GET /healthz` → `{ "status": "ok" }`

## Auth

- `POST /api/auth/login` `{ username|email, password }` → `{ token, user }`
- `POST /api/auth/register` `{ username, email, password }` → member user
- `GET /api/auth/me`

## Users (admin/operator for list; admin for mutate)

- `GET /api/users`
- `POST /api/users` `{ username, email, password, role }`
- `PUT /api/users/:id`
- `DELETE /api/users/:id`

## Vscales

- `GET /api/vscales`
- `POST /api/vscales` `{ name, description?, overlayCidr?, dns? }`
- `GET /api/vscales/:id`
- `PUT /api/vscales/:id`
- `DELETE /api/vscales/:id`
- `GET /api/vscales/:id/topology`
- `GET|POST /api/vscales/:id/members`
- `DELETE /api/vscales/:id/members/:userId`

## Hubs

- `GET /api/vscales/:id/hubs`
- `POST /api/vscales/:id/hubs` `{ name, endpoint, listenPort?, mtu? }` → includes `agentToken`
- `DELETE /api/vscales/:id/hubs/:hubId`
- `POST /api/vscales/:id/hubs/:hubId/token` → rotate token

## Devices

- `GET /api/vscales/:id/devices` (includes `online`)
- `PUT /api/vscales/:id/devices/:deviceId`
- `DELETE /api/vscales/:id/devices/:deviceId` (revoke)

## Join tokens

- `GET /api/vscales/:id/join-tokens`
- `POST /api/vscales/:id/join-tokens` `{ label?, reusable?, expiresInHours?, uses? }`
- `DELETE /api/vscales/:id/join-tokens/:tokenId`

## Routes

- `GET /api/vscales/:id/routes`
- `POST /api/vscales/:id/routes` `{ cidr, viaType, viaId, autoApprove?, force?, localCidrs? }`
- `POST /api/vscales/:id/routes/:routeId/approve` `{ force?, localCidrs? }`
- `POST /api/vscales/:id/routes/:routeId/deny`
- `DELETE /api/vscales/:id/routes/:routeId`

## Allocator

- `POST /api/allocator/check` `{ cidr, vscaleId?, localCidrs? }` → conflicts

## Stats

- `GET /api/stats` → vscales, hubs, devices online, routes, transfer totals

## Agent protocol

- `POST /api/agent/enroll` `{ token, name?, hostname?, os? }` → keys + `agentToken`
- `GET /api/agent/config` (device or hub) → interface, peers, routes
- `POST /api/agent/heartbeat` `{ transferRx, transferTx, localCidrs, hostname, os, peerStats }`
- `GET /api/agent/peers` → same as config (hub convenience)
