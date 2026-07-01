#!/usr/bin/env bash
# OWPanel stack fallback entrypoint — used when distro apt/dnf install fails.
# Strategy: panel tries system packages first; this script is fetched from GitHub (luuuunet/owpanel).
# Usage: fallback.sh nginx|redis|postgresql|mongodb|mariadb|…
set -euo pipefail

# Fix broken third-party apt lists before any install (e.g. MongoDB noble repo blocks all apt updates).
owpanel_apt_emergency_sanitize() {
  if ! command -v apt-get >/dev/null 2>&1; then
    return 0
  fi
  export DEBIAN_FRONTEND=noninteractive
  shopt -s nullglob
  local f
  for f in /etc/apt/sources.list.d/*.list; do
    if grep -q 'repo.mongodb.org' "$f" 2>/dev/null; then
      if grep -qE '/ubuntu (noble|mantic)/' "$f" 2>/dev/null; then
        sed -i 's|/ubuntu noble/mongodb-org/|/ubuntu jammy/mongodb-org/|g;s|/ubuntu mantic/mongodb-org/|/ubuntu jammy/mongodb-org/|g' "$f" 2>/dev/null || rm -f "$f"
      fi
    fi
  done
  shopt -u nullglob
  if apt-get update -qq 2>/tmp/owpanel-apt-pre.err; then
    return 0
  fi
  if grep -qE 'mongodb.org|Release file' /tmp/owpanel-apt-pre.err 2>/dev/null; then
    rm -f /etc/apt/sources.list.d/mongodb-org-*.list
    apt-get update -qq || true
  fi
  if grep -qE 'repo\.mysql\.com|EXPKEYSIG.*[Mm]ysql|mysql.*not signed' /tmp/owpanel-apt-pre.err 2>/dev/null; then
    rm -f /etc/apt/sources.list.d/*mysql* /etc/apt/sources.list.d/mysql*.list 2>/dev/null || true
    DEBIAN_FRONTEND=noninteractive apt-get remove -y --purge mysql-apt-config 2>/dev/null || true
    apt-get update -qq || true
  fi
}
owpanel_apt_emergency_sanitize

COMPONENT="${1:-}"
[[ -n "$COMPONENT" ]] || { echo "usage: $0 <component>" >&2; exit 1; }

if [[ -f "${BASH_SOURCE[0]:-}" ]]; then
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/common.sh"
else
  SCRIPT_DIR="/tmp/owpanel-stack-$$"
  mkdir -p "$SCRIPT_DIR"
  curl -fsSL --connect-timeout 30 --max-time 120 --retry 3 \
    "$(owpanel_github_stack_raw_base 2>/dev/null || echo "https://raw.githubusercontent.com/luuuunet/owpanel/main/scripts/stack")/common.sh" \
    -o "$SCRIPT_DIR/common.sh" || {
    echo "[owpanel-stack] ERROR: cannot download common.sh from GitHub" >&2
    exit 1
  }
  # shellcheck disable=SC1091
  source "$SCRIPT_DIR/common.sh"
  owpanel_download_stack_scripts "$SCRIPT_DIR" || {
    echo "[owpanel-stack] ERROR: cannot download stack scripts from GitHub" >&2
    exit 1
  }
fi

case "$COMPONENT" in
  nginx) exec bash "$SCRIPT_DIR/install-nginx.sh" ;;
  mariadb|mysql) exec bash "$SCRIPT_DIR/install-mariadb.sh" ;;
  php*)
    ver="${COMPONENT#php}"
    if [[ ${#ver} -ge 2 ]]; then
      export PHP_VERSION="${ver:0:1}.${ver:1}"
    fi
    exec bash "$SCRIPT_DIR/install-php.sh"
    ;;
  redis) exec bash "$SCRIPT_DIR/install-redis.sh" ;;
  postgresql) exec bash "$SCRIPT_DIR/install-postgresql.sh" ;;
  mongodb) exec bash "$SCRIPT_DIR/install-mongodb.sh" ;;
  apache) exec bash "$SCRIPT_DIR/install-apache.sh" ;;
  openresty) exec bash "$SCRIPT_DIR/install-openresty.sh" ;;
  docker) exec bash "$SCRIPT_DIR/install-docker.sh" ;;
  certbot) exec bash "$SCRIPT_DIR/install-certbot.sh" ;;
  memcached|fail2ban|supervisor|pureftpd|postfix|dovecot)
    export GENERIC_KEY="$COMPONENT"
    exec bash "$SCRIPT_DIR/install-generic.sh"
    ;;
  *)
    echo "unknown component: $COMPONENT" >&2
    exit 1
    ;;
esac
