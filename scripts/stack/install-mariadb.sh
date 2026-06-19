#!/usr/bin/env bash
# Install MariaDB (preferred over Oracle MySQL on small Debian VPS).
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/common.sh"

require_root
detect_os

for svc in mariadb mysql mysqld; do
  if service_active "$svc"; then
    log "database service already running ($svc)"
    exit 0
  fi
done

ensure_prereqs

case "$PKG" in
  apt)
    export DEBIAN_FRONTEND=noninteractive
    debconf-set-selections <<'EOF' || true
mariadb-server mariadb-server/root_password password owpanel
mariadb-server mariadb-server/root_password_again password owpanel
EOF
    if apt_install mariadb-server 2>/dev/null; then
      :
    elif apt_install default-mysql-server 2>/dev/null; then
      log "installed default-mysql-server (MariaDB-compatible)"
    else
      log "trying MariaDB upstream repo …"
      ensure_codename
      curl -fsSL --connect-timeout 30 --max-time 120 --retry 3 \
        https://mariadb.org/mariadb_release_signing_key.pgp \
        | gpg --dearmor -o /usr/share/keyrings/mariadb-keyring.gpg
      cat > /etc/apt/sources.list.d/mariadb-owpanel.list <<EOF
deb [signed-by=/usr/share/keyrings/mariadb-keyring.gpg] https://mirrors.aliyun.com/mariadb/mariadb-10.11/repo/debian ${OS_CODENAME} main
EOF
      apt_update
      apt_install mariadb-server || apt_install mariadb-server-10.11
    fi
    tune_mariadb_lowmem
    ;;
  dnf|yum)
    $PKG install -y mariadb-server 2>/dev/null || $PKG install -y mysql-server
    ;;
esac

for svc in mariadb mysql mysqld; do
  if systemctl list-unit-files "${svc}.service" >/dev/null 2>&1; then
    enable_start "$svc"
    log "database service started ($svc)"
    exit 0
  fi
done

die "MariaDB installed but no service unit found"
