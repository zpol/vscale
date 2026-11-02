#!/usr/bin/env bash
set -euo pipefail

HOST="${VSCALE_PROD_HOST:-198.46.253.14}"
USER="${VSCALE_PROD_USER:-root}"
REMOTE="/opt/vscale"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOMAIN="${DOMAIN:-vscale.0ffsec.com}"
EMAIL="${CERTBOT_EMAIL:-admin@0ffsec.com}"

echo "==> sync to ${USER}@${HOST}:${REMOTE}"
ssh "${USER}@${HOST}" "mkdir -p ${REMOTE} /var/www/certbot"
rsync -az --delete \
  --exclude '.git' \
  --exclude 'web/node_modules' \
  --exclude 'web/build' \
  --exclude 'dist' \
  --exclude 'wg-manager/frontend/node_modules' \
  "${ROOT}/" "${USER}@${HOST}:${REMOTE}/"

echo "==> write env + build/up"
ssh "${USER}@${HOST}" bash -s <<EOF
set -euo pipefail
cd ${REMOTE}/deploy/prod
if [[ ! -f .env ]]; then
  cat > .env <<'ENV'
DB_PASSWORD=$(openssl rand -hex 16)
JWT_SECRET=$(openssl rand -hex 32)
DEFAULT_ADMIN_USER=admin
DEFAULT_ADMIN_PASSWORD=cameloso787
DEFAULT_ADMIN_EMAIL=admin@vscale.local
PUBLIC_URL=https://${DOMAIN}
CORS_ORIGINS=https://${DOMAIN}
HUB_ENDPOINT=198.46.253.14
HUB_LISTEN_PORT=51820
DOMAIN=${DOMAIN}
CERTBOT_EMAIL=${EMAIL}
ENV
else
  # keep secrets, refresh admin password + public URL
  sed -i 's|^DEFAULT_ADMIN_PASSWORD=.*|DEFAULT_ADMIN_PASSWORD=cameloso787|' .env
  sed -i 's|^PUBLIC_URL=.*|PUBLIC_URL=https://${DOMAIN}|' .env
  sed -i 's|^CORS_ORIGINS=.*|CORS_ORIGINS=https://${DOMAIN}|' .env
fi

docker compose -f docker-compose.yml build
docker compose -f docker-compose.yml up -d postgres control-plane web

# nginx site (HTTP first for certbot if needed)
install -m 644 ${REMOTE}/deploy/prod/nginx-vscale.conf /etc/nginx/sites-available/vscale.0ffsec.com
# temporary HTTP-only until cert exists
if [[ ! -d /etc/letsencrypt/live/${DOMAIN} ]]; then
  cat > /etc/nginx/sites-available/vscale.0ffsec.com <<'NGX'
server {
    listen 80;
    listen [::]:80;
    server_name vscale.0ffsec.com;
    location /.well-known/acme-challenge/ { root /var/www/certbot; }
    location /api/ {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host \$host;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Real-IP \$remote_addr;
    }
    location /healthz { proxy_pass http://127.0.0.1:18080/healthz; }
    location / { proxy_pass http://127.0.0.1:13000; proxy_set_header Host \$host; }
}
NGX
fi
ln -sfn /etc/nginx/sites-available/vscale.0ffsec.com /etc/nginx/sites-enabled/vscale.0ffsec.com
nginx -t && systemctl reload nginx

if [[ ! -d /etc/letsencrypt/live/${DOMAIN} ]]; then
  certbot --nginx -d ${DOMAIN} --non-interactive --agree-tos -m ${EMAIL} --redirect
else
  # restore full SSL conf after renewals
  install -m 644 ${REMOTE}/deploy/prod/nginx-vscale.conf /etc/nginx/sites-available/vscale.0ffsec.com
  nginx -t && systemctl reload nginx
fi

# certbot timer usually enabled by package; ensure it
systemctl enable --now certbot.timer 2>/dev/null || true
systemctl list-timers | grep -i certbot || true

docker compose -f docker-compose.yml ps
curl -fsS http://127.0.0.1:18080/healthz
EOF

echo "==> bootstrap hub on VPS"
ssh "${USER}@${HOST}" bash -s <<'EOF'
set -euo pipefail
cd /opt/vscale
source deploy/prod/.env
API=http://127.0.0.1:18080
# wait API
for i in $(seq 1 30); do curl -fsS "$API/healthz" >/dev/null && break; sleep 2; done
TOKEN=$(curl -fsS -X POST "$API/api/auth/login" -H 'Content-Type: application/json' \
  -d "{\"username\":\"$DEFAULT_ADMIN_USER\",\"password\":\"$DEFAULT_ADMIN_PASSWORD\"}" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
AUTH=(-H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json')

# ensure vscale
VSID=$(curl -fsS "$API/api/vscales" -H "Authorization: Bearer $TOKEN" \
  | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d[0]["ID"] if d else "")')
if [[ -z "$VSID" ]]; then
  VSID=$(curl -fsS -X POST "$API/api/vscales" "${AUTH[@]}" \
    -d '{"name":"production","description":"public hub"}' \
    | python3 -c 'import sys,json; print(json.load(sys.stdin)["vscale"]["ID"])')
fi

HUBS=$(curl -fsS "$API/api/vscales/$VSID/hubs" -H "Authorization: Bearer $TOKEN")
HUBID=$(echo "$HUBS" | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d[0]["ID"] if d else "")')
if [[ -z "$HUBID" ]]; then
  HUB=$(curl -fsS -X POST "$API/api/vscales/$VSID/hubs" "${AUTH[@]}" \
    -d "{\"name\":\"hub-prod\",\"endpoint\":\"$HUB_ENDPOINT\",\"listenPort\":${HUB_LISTEN_PORT:-51820}}")
  HUB_TOKEN=$(echo "$HUB" | python3 -c 'import sys,json; print(json.load(sys.stdin)["agentToken"])')
  HUBID=$(echo "$HUB" | python3 -c 'import sys,json; print(json.load(sys.stdin)["hub"]["ID"])')
else
  HUB_TOKEN=$(curl -fsS -X POST "$API/api/vscales/$VSID/hubs/$HUBID/token" -H "Authorization: Bearer $TOKEN" \
    | python3 -c 'import sys,json; print(json.load(sys.stdin)["agentToken"])')
fi

mkdir -p /opt/vscale/hub-data
cat > /opt/vscale/hub-data/state.json <<STATE
{
  "serverUrl": "http://127.0.0.1:18080",
  "agentToken": "$HUB_TOKEN",
  "role": "hub",
  "up": false
}
STATE

# run hub privileged with host network
docker rm -f vscale-hub 2>/dev/null || true
docker build -t vscale-agent:local -f /opt/vscale/deploy/Dockerfile.agent /opt/vscale
docker run -d --name vscale-hub --restart unless-stopped --privileged \
  --network host \
  --device /dev/net/tun \
  -e VSCALE_STATE=/data/state.json \
  -v /opt/vscale/hub-data:/data \
  vscale-agent:local daemon

sleep 4
docker logs vscale-hub --tail 20
ss -ulnp | grep 51820 || true
ip link show vscale0 || true
echo "VSID=$VSID HUBID=$HUBID"
EOF

echo "==> done"
echo "UI:  https://${DOMAIN}"
echo "API: https://${DOMAIN}/api"
echo "Login: admin / cameloso787"
echo "Hub UDP: 198.46.253.14:51820"
