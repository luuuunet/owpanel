#!/usr/bin/env bash
# Install Nginx via apt, falling back to nginx.org official repo on Debian/Ubuntu.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/common.sh"

require_root
detect_os

if command -v nginx >/dev/null 2>&1; then
  log "nginx already installed"
  enable_start nginx
  exit 0
fi

ensure_prereqs

case "$PKG" in
  apt)
    if apt_install nginx 2>/dev/null; then
      enable_start nginx
      log "nginx installed from default apt"
      exit 0
    fi
    log "default apt nginx failed, trying nginx.org repo …"
    ensure_codename
    install -d -m 0755 /usr/share/keyrings
    curl -fsSL --connect-timeout 30 --max-time 120 --retry 3 \
      https://nginx.org/keys/nginx_signing.key \
      | gpg --dearmor -o /usr/share/keyrings/nginx-archive-keyring.gpg
    distro="$OS_ID"
    [[ "$distro" == "debian" || "$distro" == "ubuntu" ]] || distro="debian"
    cat > /etc/apt/sources.list.d/nginx-owpanel.list <<EOF
deb [signed-by=/usr/share/keyrings/nginx-archive-keyring.gpg] https://nginx.org/packages/${distro}/ ${OS_CODENAME} nginx
EOF
    apt_update
    apt_install nginx
    enable_start nginx
    log "nginx installed from nginx.org repo"
    ;;
  dnf|yum)
    $PKG install -y nginx
    enable_start nginx
    log "nginx installed from $PKG"
    ;;
  *)
    die "unsupported package manager: $PKG"
    ;;
esac
