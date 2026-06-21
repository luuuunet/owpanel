#!/usr/bin/env bash
# Copy scripts/stack into the Go embed tree before building owpanel.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/backend/internal/stackscripts/stack"
rm -rf "$DEST"
cp -a "$ROOT/scripts/stack" "$DEST"
find "$DEST" -name '*.sh' -exec sed -i 's/\r$//' {} \; 2>/dev/null || \
  find "$DEST" -name '*.sh' -exec perl -pi -e 's/\r\n/\n/g' {} \;
find "$DEST" -name '*.sh' -exec chmod +x {} \;
echo "[sync-stack-embed] synced $(find "$DEST" -name '*.sh' | wc -l | tr -d ' ') scripts -> $DEST"
