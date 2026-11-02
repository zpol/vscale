#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/dist"
mkdir -p "$OUT"
export CGO_ENABLED=0

build() {
  local os="$1" arch="$2" ext="${3:-}"
  local name="vscale_${os}_${arch}${ext}"
  echo "building ${name}"
  GOOS="$os" GOARCH="$arch" go build -o "${OUT}/${name}" "${ROOT}/cmd/vscale"
}

build linux amd64
build linux arm64
build darwin amd64
build darwin arm64
build windows amd64 .exe
echo "artifacts in ${OUT}"
ls -la "$OUT"
