#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
# shellcheck disable=SC1091
source "$SCRIPT_DIR/common.sh"

require_root
detect_os

if systemctl is-active --quiet mongod 2>/dev/null; then
  log "mongodb already running (systemd mongod)"
  exit 0
fi
if command -v docker >/dev/null 2>&1 && docker ps --format '{{.Names}}' 2>/dev/null | grep -qx 'owpanel-mongodb'; then
  log "mongodb already running (docker owpanel-mongodb)"
  exit 0
fi

ensure_prereqs

install_mongodb_org() {
  local ver="${1:-7.0}"
  rm -f "/etc/apt/sources.list.d/mongodb-org-${ver}.list"
  if ! setup_mongodb_repo "$ver"; then
    log "MongoDB ${ver} repo setup failed"
    return 1
  fi
  if try_apt_retry mongodb-org mongodb-org-server mongodb-org-database mongodb-mongosh; then
    return 0
  fi
  if try_apt_retry mongodb-org; then
    return 0
  fi
  log "MongoDB ${ver} packages not available from apt"
  rm -f "/etc/apt/sources.list.d/mongodb-org-${ver}.list"
  return 1
}

install_mongodb_docker() {
  log "trying MongoDB via Docker (mongo:7) …"
  if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
    log "Docker not ready — installing Docker first …"
    bash "$SCRIPT_DIR/install-docker.sh" || return 1
  fi
  docker rm -f owpanel-mongodb 2>/dev/null || true
  docker volume create owpanel-mongodb-data >/dev/null 2>&1 || true
  docker run -d --name owpanel-mongodb --restart unless-stopped \
    -p 127.0.0.1:27017:27017 \
    -v owpanel-mongodb-data:/data/db \
    mongo:7.0 --bind_ip_all
  local i
  for i in $(seq 1 30); do
    if docker exec owpanel-mongodb mongosh --quiet --eval 'db.adminCommand({ping:1})' >/dev/null 2>&1; then
      log "mongodb running in Docker on 127.0.0.1:27017 (container: owpanel-mongodb)"
      return 0
    fi
    sleep 2
  done
  docker logs owpanel-mongodb 2>&1 | tail -20 >&2 || true
  return 1
}

case "$PKG" in
  apt)
    log "步骤 1：尝试 MongoDB 官方 apt 源 (mongodb-org) …"
    for ver in 7.0 6.0 8.0; do
      if install_mongodb_org "$ver"; then
        enable_start mongod || systemctl start mongod || true
        log "mongodb ${ver} installed from official repo"
        exit 0
      fi
    done
    log "步骤 2：官方 apt 源均失败，使用 Docker 运行 MongoDB …"
    install_mongodb_docker || die "mongodb install failed (apt repos and Docker fallback)"
    exit 0
    ;;
  dnf|yum)
    cat > /etc/yum.repos.d/mongodb-org-7.0.repo <<'EOF'
[mongodb-org-7.0]
name=MongoDB Repository
baseurl=https://repo.mongodb.org/yum/redhat/$releasever/mongodb-org/7.0/x86_64/
gpgcheck=1
enabled=1
gpgkey=https://pgp.mongodb.com/server-7.0.asc
EOF
    if ! $PKG install -y mongodb-org; then
      install_mongodb_docker || die "mongodb install failed"
      exit 0
    fi
    enable_start mongod
    ;;
  *) die "unsupported package manager: $PKG" ;;
esac
log "mongodb installed"
