#!/usr/bin/env bash
# One-liner home install:
#   curl -fsSL https://raw.githubusercontent.com/zpol/vscale/main/scripts/install.sh | sudo bash
set -euo pipefail

REPO="${VSCALE_REPO:-zpol/vscale}"
API="https://api.github.com/repos/${REPO}/releases/latest"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

need_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "Re-run with sudo:" >&2
    echo "  curl -fsSL https://raw.githubusercontent.com/zpol/vscale/main/scripts/install.sh | sudo bash" >&2
    exit 1
  fi
}

detect_pkg() {
  if command -v dpkg >/dev/null 2>&1; then
    echo deb
  elif command -v rpm >/dev/null 2>&1; then
    echo rpm
  else
    echo bin
  fi
}

arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo amd64 ;;
    aarch64|arm64) echo arm64 ;;
    *) echo "unsupported arch $(uname -m)" >&2; exit 1 ;;
  esac
}

install_wireguard() {
  if command -v wg-quick >/dev/null 2>&1; then
    return 0
  fi
  echo "Installing wireguard-tools…"
  if command -v apt-get >/dev/null 2>&1; then
    apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq wireguard wireguard-tools
  elif command -v dnf >/dev/null 2>&1; then
    dnf install -y wireguard-tools
  elif command -v yum >/dev/null 2>&1; then
    yum install -y wireguard-tools
  else
    echo "Install wireguard-tools manually, then re-run." >&2
    exit 1
  fi
}

need_root
A="$(arch)"
KIND="$(detect_pkg)"
install_wireguard

echo "Fetching latest release from ${REPO}…"
if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

JSON="$TMP/release.json"
curl -fsSL "$API" -o "$JSON"

asset_url() {
  # $1 = substring to match in asset name
  python3 - "$JSON" "$1" <<'PY' 2>/dev/null || true
import json,sys
j=json.load(open(sys.argv[1]))
needle=sys.argv[2]
for a in j.get("assets") or []:
    name=a.get("name") or ""
    if needle in name:
        print(a["browser_download_url"]); break
PY
}

case "$KIND" in
  deb)
    URL="$(asset_url "_${A}.deb")"
    if [ -z "$URL" ]; then URL="$(asset_url "linux_${A}.deb")"; fi
    if [ -z "$URL" ]; then URL="$(asset_url ".deb")"; fi
    if [ -n "$URL" ]; then
      echo "Downloading $URL"
      curl -fsSL "$URL" -o "$TMP/vscale.deb"
      dpkg -i "$TMP/vscale.deb" || apt-get install -f -y
      echo
      echo "OK. Run:  vscale setup"
      exit 0
    fi
    ;;
  rpm)
    URL="$(asset_url "_${A}.rpm")"
    if [ -z "$URL" ]; then URL="$(asset_url "linux_${A}.rpm")"; fi
    if [ -n "$URL" ]; then
      echo "Downloading $URL"
      curl -fsSL "$URL" -o "$TMP/vscale.rpm"
      if command -v dnf >/dev/null 2>&1; then dnf install -y "$TMP/vscale.rpm"
      else rpm -Uvh "$TMP/vscale.rpm"; fi
      echo
      echo "OK. Run:  vscale setup"
      exit 0
    fi
    ;;
esac

# Fallback: raw binary from release archive / asset
URL="$(asset_url "vscale_linux_${A}.tar.gz")"
if [ -z "$URL" ]; then URL="$(asset_url "Linux_${A}.tar.gz")"; fi
if [ -z "$URL" ]; then
  # direct binary asset naming from scripts/build-agent.sh style
  URL="$(asset_url "vscale_linux_${A}")"
fi
if [ -z "$URL" ]; then
  echo "No suitable release asset found. Build from source:" >&2
  echo "  go install github.com/zpol/vscale/cmd/vscale@latest" >&2
  exit 1
fi

echo "Downloading $URL"
curl -fsSL "$URL" -o "$TMP/asset"
if file "$TMP/asset" | grep -qi 'gzip\|tar'; then
  tar -C "$TMP" -xzf "$TMP/asset"
  BIN="$(find "$TMP" -type f -name vscale | head -1)"
else
  BIN="$TMP/asset"
fi
install -m 755 "$BIN" /usr/bin/vscale
echo
echo "OK. Run:  vscale setup"
