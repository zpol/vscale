#!/usr/bin/env bash
set -euo pipefail

API="${VSCALE_API:-http://192.168.68.113:30808}"
USER="${VSCALE_ADMIN_USER:-admin}"
PASS="${VSCALE_ADMIN_PASSWORD:-admin123}"
AGENT_BIN="${AGENT_BIN:-$(cd "$(dirname "$0")/.." && pwd)/dist/vscale_linux_amd64}"

echo "==> health"
curl -fsS "${API}/healthz" | grep -q ok

echo "==> login"
TOKEN=$(curl -fsS -X POST "${API}/api/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${USER}\",\"password\":\"${PASS}\"}" | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

auth=(-H "Authorization: Bearer ${TOKEN}" -H 'Content-Type: application/json')

echo "==> create vscale"
VS=$(curl -fsS -X POST "${API}/api/vscales" "${auth[@]}" \
  -d '{"name":"e2e-'"$(date +%s)"'","description":"e2e"}')
VSID=$(echo "$VS" | python3 -c 'import sys,json; print(json.load(sys.stdin)["vscale"]["ID"])')
CIDR=$(echo "$VS" | python3 -c 'import sys,json; print(json.load(sys.stdin)["vscale"]["overlayCidr"])')
echo "vscale ${VSID} overlay ${CIDR}"

echo "==> create hub"
HUB=$(curl -fsS -X POST "${API}/api/vscales/${VSID}/hubs" "${auth[@]}" \
  -d '{"name":"hub-e2e","endpoint":"192.168.68.113","listenPort":51820}')
HUB_TOKEN=$(echo "$HUB" | python3 -c 'import sys,json; print(json.load(sys.stdin)["agentToken"])')
echo "hub token ok"

echo "==> join token"
JT=$(curl -fsS -X POST "${API}/api/vscales/${VSID}/join-tokens" "${auth[@]}" \
  -d '{"label":"e2e","reusable":true,"uses":10}')
JOIN=$(echo "$JT" | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')

echo "==> stats"
curl -fsS "${API}/api/stats" -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool

echo "==> topology"
curl -fsS "${API}/api/vscales/${VSID}/topology" -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool

if [[ -x "${AGENT_BIN}" ]]; then
  export VSCALE_STATE=/tmp/vscale-e2e-state.json
  rm -f "$VSCALE_STATE"
  echo "==> agent enroll"
  "${AGENT_BIN}" join "${JOIN}" --server "${API}" --name e2e-client
  echo "==> agent status"
  "${AGENT_BIN}" status
  echo "==> agent fetch config (up may need privileges/wg-quick)"
  "${AGENT_BIN}" up || true
  "${AGENT_BIN}" status
else
  echo "skip agent binary tests (${AGENT_BIN} missing); run scripts/build-agent.sh first"
fi

echo "E2E API OK"
