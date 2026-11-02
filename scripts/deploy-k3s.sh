#!/usr/bin/env bash
set -euo pipefail

HOST="${VSCALE_K3S_HOST:-192.168.68.113}"
USER="${VSCALE_K3S_USER:-nkn}"
REMOTE_DIR="${VSCALE_REMOTE_DIR:-/home/${USER}/vscale}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "==> syncing to ${USER}@${HOST}:${REMOTE_DIR}"
rsync -az --delete \
  --exclude '.git' \
  --exclude 'wg-manager/frontend/node_modules' \
  --exclude 'web/node_modules' \
  --exclude 'web/build' \
  "${ROOT}/" "${USER}@${HOST}:${REMOTE_DIR}/"

echo "==> building images on cluster node"
ssh "${USER}@${HOST}" bash -s <<EOF
set -euo pipefail
cd ${REMOTE_DIR}
docker build -t vscale-control-plane:local -f deploy/Dockerfile.control-plane .
docker build -t vscale-web:local --build-arg REACT_APP_API_URL=http://${HOST}:30808 -f deploy/Dockerfile.web .
docker build -t vscale-agent:local -f deploy/Dockerfile.agent .
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/postgres.yaml
kubectl apply -f deploy/k8s/control-plane.yaml
kubectl apply -f deploy/k8s/web.yaml
kubectl -n vscale rollout restart deploy/control-plane deploy/web || true
kubectl -n vscale rollout status deploy/postgres --timeout=120s
kubectl -n vscale rollout status deploy/control-plane --timeout=180s
kubectl -n vscale rollout status deploy/web --timeout=180s
kubectl -n vscale get pods,svc
EOF

echo "==> control plane: http://${HOST}:30808"
echo "==> web UI:        http://${HOST}:30300"
echo "==> admin / admin123 (change after first login)"
