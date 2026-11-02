#!/usr/bin/env bash
# Build a .deb for the vscale agent without requiring nfpm/goreleaser.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${VERSION:-0.1.0}"
ARCH_RAW="$(uname -m)"
case "$ARCH_RAW" in
  x86_64|amd64) DEB_ARCH=amd64; GOARCH=amd64 ;;
  aarch64|arm64) DEB_ARCH=arm64; GOARCH=arm64 ;;
  *) echo "unsupported arch: $ARCH_RAW" >&2; exit 1 ;;
esac

OUT="${ROOT}/dist"
STAGE="${OUT}/deb-root"
PKG_NAME="vscale_${VERSION}_${DEB_ARCH}"
PKG_DIR="${OUT}/${PKG_NAME}"

rm -rf "$STAGE" "$PKG_DIR"
mkdir -p "$STAGE/usr/bin" \
  "$STAGE/usr/share/doc/vscale" \
  "$STAGE/DEBIAN"

echo "building agent ($GOARCH)…"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -ldflags "-s -w -X main.version=${VERSION}" \
  -o "$STAGE/usr/bin/vscale" "${ROOT}/cmd/vscale"

cp "${ROOT}/packaging/debian/copyright" "$STAGE/usr/share/doc/vscale/copyright"
cp "${ROOT}/docs/user-guide.md" "$STAGE/usr/share/doc/vscale/user-guide.md"
gzip -n -c9 "${ROOT}/packaging/debian/changelog" > "$STAGE/usr/share/doc/vscale/changelog.gz"

SIZE_KB="$(du -sk "$STAGE" | awk '{print $1}')"

cat > "$STAGE/DEBIAN/control" <<EOF
Package: vscale
Version: ${VERSION}
Section: net
Priority: optional
Architecture: ${DEB_ARCH}
Depends: wireguard-tools
Maintainer: zpol <zpol@users.noreply.github.com>
Installed-Size: ${SIZE_KB}
Homepage: https://github.com/zpol/vscale
Description: Easy hub-spoke WireGuard VPN agent
 vscale connects home/office PCs to a self-hosted hub-spoke VPN.
 Install, run "vscale setup", and you are on the overlay network.
EOF

cat > "$STAGE/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
cat <<'EOM'

  vscale installed.

  Next (ask your admin for a join token):

    vscale setup
    # or:
    vscale join <TOKEN> --server https://your-server.example
    vscale up

  Docs: https://github.com/zpol/vscale

EOM
EOF
chmod 755 "$STAGE/DEBIAN/postinst" "$STAGE/usr/bin/vscale"

mkdir -p "$OUT"
dpkg-deb --root-owner-group --build "$STAGE" "${OUT}/${PKG_NAME}.deb"
echo "built ${OUT}/${PKG_NAME}.deb"
dpkg-deb -I "${OUT}/${PKG_NAME}.deb"
ls -lh "${OUT}/${PKG_NAME}.deb"
