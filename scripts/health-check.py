#!/usr/bin/env python3
"""SSH + API smoke checks for OWPanel production server."""
from __future__ import annotations

import json
import re
import subprocess
import sys
from pathlib import Path

import paramiko

ENV = Path(__file__).parent / "deploy.env"
cfg = {
    k: v
    for line in ENV.read_text(encoding="utf-8").splitlines()
    if "=" in line and not line.strip().startswith("#")
    for k, v in [line.strip().split("=", 1)]
}

host = cfg["DEPLOY_HOST"]
port = int(cfg.get("OWPANEL_PORT", "8888"))
install = cfg.get("INSTALL_DIR", "/opt/owpanel")

c = paramiko.SSHClient()
c.set_missing_host_key_policy(paramiko.AutoAddPolicy())
c.connect(host, username=cfg["DEPLOY_USER"], password=cfg["DEPLOY_PASSWORD"], timeout=30, banner_timeout=60)


def run(cmd: str) -> str:
    _, o, _ = c.exec_command(cmd, timeout=30)
    return o.read().decode("utf-8", "replace").strip()


prefix = ""
for line in run(f"{install}/op info 2>/dev/null").splitlines():
    m = re.search(rf":{port}/([^/\s]+)", line)
    if m:
        prefix = m.group(1)
        break
if not prefix:
    prefix = run(
        f"sqlite3 {install}/data/panel.db \"SELECT value FROM panel_settings WHERE key='panel_safe_path' LIMIT 1\""
    ).strip().strip("/") or "login"

checks = [
    ("panel", "systemctl is-active owpanel"),
    ("health", f"curl -sf http://127.0.0.1:{port}/{prefix}/health"),
    ("bootstrap", f"curl -sf http://127.0.0.1:{port}/{prefix}/api/v1/auth/bootstrap"),
    ("whitelist", f'sqlite3 {install}/data/panel.db "SELECT value FROM panel_settings WHERE key=\'panel_ip_whitelist_enabled\'"'),
    ("memory", "free -h | head -2"),
    ("docker", "docker info >/dev/null 2>&1 && echo ok || echo not_ok"),
    ("nginx", "systemctl is-active nginx 2>/dev/null || echo inactive"),
]
failed = 0
for name, cmd in checks:
    out = run(cmd)
    ok = out and "inactive" not in out.lower() and "not_ok" not in out
    if name in ("nginx", "whitelist") or not ok:
        status = out or "(empty)"
    else:
        status = "ok"
    print(f"[{name}] {status}")
    if name in ("panel", "health", "bootstrap") and not ok:
        failed += 1

c.close()

# Full API smoke suite
smoke = subprocess.run([sys.executable, str(Path(__file__).parent / "api-smoke.py")])
sys.exit(failed + smoke.returncode)
