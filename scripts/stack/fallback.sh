#!/usr/bin/env bash
# OWPanel stack fallback entrypoint — used by appstore when apt install fails.
# Usage: fallback.sh nginx|mariadb|mysql|php83|php82|…
# Remote: curl -fsSL https://raw.githubusercontent.com/luuuunet/owpanel/main/scripts/stack/fallback.sh | bash -s -- nginx
set -euo pipefail

COMPONENT="${1:-}"
[[ -n "$COMPONENT" ]] || { echo "usage: $0 nginx|mariadb|mysql|php83|…" >&2; exit 1; }

if [[ -f "${BASH_SOURCE[0]:-}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
else
  SCRIPT_DIR="/tmp/owpanel-stack-$$"
  mkdir -p "$SCRIPT_DIR"
  BASE="${OWPANEL_STACK_BASE:-https://raw.githubusercontent.com/luuuunet/owpanel/main/scripts/stack}"
  for f in common.sh install-nginx.sh install-mariadb.sh install-php.sh; do
    curl -fsSL --connect-timeout 30 --max-time 120 --retry 3 \
      "${BASE}/${f}" -o "${SCRIPT_DIR}/${f}"
    chmod +x "${SCRIPT_DIR}/${f}"
  done
fi

case "$COMPONENT" in
  nginx)
    exec bash "$SCRIPT_DIR/install-nginx.sh"
    ;;
  mariadb|mysql)
    exec bash "$SCRIPT_DIR/install-mariadb.sh"
    ;;
  php*)
    ver="${COMPONENT#php}"
    if [[ ${#ver} -ge 2 ]]; then
      export PHP_VERSION="${ver:0:1}.${ver:1}"
    fi
    exec bash "$SCRIPT_DIR/install-php.sh"
    ;;
  *)
    echo "unknown component: $COMPONENT" >&2
    exit 1
    ;;
esac
