#!/bin/sh
# Builds poltio.mcpb — the double-click installer bundle for Claude Desktop.
# Usage: scripts/build-mcpb.sh [version]
# Must run on macOS: lipo is needed for the universal (Intel + Apple Silicon) build.
set -eu

VERSION="${1:-0.0.0}"
VERSION="${VERSION#v}"
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
OUT="$ROOT/dist/mcpb"

rm -rf "$OUT"
mkdir -p "$OUT/server"

build() {
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$3" "$ROOT"
}

build darwin arm64 "$OUT/server/darwin-arm64"
build darwin amd64 "$OUT/server/darwin-amd64"
lipo -create -output "$OUT/server/poltio-mcp-server" \
  "$OUT/server/darwin-arm64" "$OUT/server/darwin-amd64"
rm "$OUT/server/darwin-arm64" "$OUT/server/darwin-amd64"

build windows amd64 "$OUT/server/poltio-mcp-server.exe"

# The binary is unsigned (no Apple Developer ID), so macOS Gatekeeper SIGKILLs it
# silently once the downloaded .mcpb has been quarantined. This wrapper clears the
# quarantine flag from our own binary before exec'ing it; it is a no-op when the
# flag is absent. Shell scripts are not subject to the Mach-O execute check, which
# is why the wrapper itself survives.
# ponytail: replace with codesign + notarytool if a Developer ID ever exists.
cat >"$OUT/server/launch.sh" <<'EOF'
#!/bin/sh
dir=$(dirname "$0")
xattr -dr com.apple.quarantine "$dir/poltio-mcp-server" 2>/dev/null || true
exec "$dir/poltio-mcp-server" "$@"
EOF
chmod +x "$OUT/server/launch.sh"

# Linux build would break the other arch. Linux users have the from-source path.

sed "s/\"version\": \"0.0.0\"/\"version\": \"$VERSION\"/" \
  "$ROOT/manifest.json" >"$OUT/manifest.json"
grep -q "\"version\": \"$VERSION\"" "$OUT/manifest.json" ||
  {
    echo "failed to stamp version into manifest.json" >&2
    exit 1
  }

(cd "$OUT" && zip -qr "$ROOT/dist/poltio.mcpb" manifest.json server)
echo "built $ROOT/dist/poltio.mcpb (version $VERSION)"
