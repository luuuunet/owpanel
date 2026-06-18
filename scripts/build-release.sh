#!/usr/bin/env bash
# Build release packages for Linux (amd64/arm64) and Windows (amd64)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$ROOT/dist"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"

log() { echo "[build] $*"; }

build_one() {
  local goos="$1" goarch="$2" ext="$3" name="$4"
  local dir="$OUT/$name"
  mkdir -p "$dir"
  log "Building $goos/$goarch -> $dir"
  (cd "$ROOT/backend" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$dir/open-panel$ext" ./cmd/server)
  (cd "$ROOT/backend" && GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -ldflags="-s -w" -o "$dir/op$ext" ./cmd/op)
  rm -rf "$dir/web"
  cp -a "$ROOT/backend/web" "$dir/web"
  mkdir -p "$dir/data"
  cat > "$dir/README.txt" <<EOF
Open Panel $VERSION ($goos/$goarch)
1. Set OPEN_PANEL_DATA to ./data (or use install script)
2. Run ./open-panel$ext  or use scripts/install.sh / install.ps1
Default: http://HOST:8888
First login: admin / random password in data/INITIAL_CREDENTIALS.txt (or server log)
EOF
  (cd "$OUT" && tar -czf "${name}.tar.gz" "$name")
  log "Package: $OUT/${name}.tar.gz"
}

log "Building frontend..."
(cd "$ROOT/frontend" && npm ci && npm run build)

build_one linux amd64 "" "open-panel-linux-amd64"
build_one linux arm64 "" "open-panel-linux-arm64"
build_one windows amd64 ".exe" "open-panel-windows-amd64"

log "Done. Artifacts in $OUT"
