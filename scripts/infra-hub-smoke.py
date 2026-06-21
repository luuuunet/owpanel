#!/usr/bin/env python3
"""Smoke-test Infra Hub API endpoints on remote panel."""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

import paramiko

ROOT = Path(__file__).parent.parent
HOST = "198.199.120.139"
USER = "root"
PASSWORD = "Wuyfieng0Wuyifeng"
PORT = 8888
INSTALL = "/opt/owpanel"
TEST_PWD = "SmokeTest123!"
RESET_BIN = ROOT / "dist" / "reset-admin-linux-amd64"

ENDPOINTS = [
    "/infra-hub/overview",
    "/infra-hub/llmops",
    "/infra-hub/dataops",
    "/infra-hub/aiops",
    "/infra-hub/secops",
    "/infra-hub/orchestration",
    "/infra-hub/vector",
    "/infra-hub/metrics",
    "/infra-hub/weights",
    "/infra-hub/security",
    "/infra-hub/storage",
    "/data-platform/overview",
]


def main() -> int:
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(HOST, username=USER, password=PASSWORD, timeout=30, banner_timeout=60)

    def run(cmd: str, timeout: int = 90) -> str:
        _, o, e = ssh.exec_command(cmd, timeout=timeout)
        out = o.read().decode("utf-8", "replace").strip()
        err = e.read().decode("utf-8", "replace").strip()
        return out or err

    svc = ""
    for _ in range(24):
        svc = run("systemctl is-active owpanel")
        if svc == "active":
            break
        import time
        time.sleep(5)
    print(f"owpanel service: {svc}")
    if svc != "active":
        ssh.close()
        return 1

    # Optional: reset admin for smoke (skip by default — slow on small VPS)
    if RESET_BIN.exists() and False:
        sftp = ssh.open_sftp()
        sftp.put(str(RESET_BIN), "/tmp/reset-admin")
        sftp.chmod("/tmp/reset-admin", 0o755)
        sftp.close()
        try:
            run(f"OWPANEL_DATA={INSTALL}/data /tmp/reset-admin '{TEST_PWD}'", timeout=120)
        except Exception as ex:
            print(f"reset-admin skipped: {ex}")

    prefix = "login"
    info = run(f"{INSTALL}/op info 2>/dev/null || true")
    for line in info.splitlines():
        m = re.search(rf":{PORT}/([^/\s]+)", line)
        if m:
            prefix = m.group(1)
            break
    if prefix == "login":
        prefix = (
            run(
                f"sqlite3 {INSTALL}/data/panel.db "
                "\"SELECT value FROM panel_settings WHERE key='panel_safe_path' LIMIT 1\""
            )
            .strip()
            .strip("/")
            or "login"
        )
    print(f"safe path prefix: {prefix}")

    base = f"http://127.0.0.1:{PORT}/{prefix}/api/v1"
    login_raw = run(
        f"curl -s -X POST -H 'Content-Type: application/json' "
        f"-d '{{\"username\":\"admin\",\"password\":\"{TEST_PWD}\"}}' '{base}/auth/login'"
    )
    try:
        login = json.loads(login_raw)
    except json.JSONDecodeError:
        print("login failed:", login_raw[:300])
        ssh.close()
        return 1

    token = (login.get("data") or {}).get("token", "")
    if not token:
        for pwd in ("admin", "Wuyfieng0Wuyifeng", TEST_PWD):
            login_raw = run(
                f"curl -s -X POST -H 'Content-Type: application/json' "
                f"-d '{{\"username\":\"admin\",\"password\":\"{pwd}\"}}' '{base}/auth/login'"
            )
            try:
                login = json.loads(login_raw)
                token = (login.get("data") or {}).get("token", "")
                if token:
                    print(f"login ok with password attempt")
                    break
            except json.JSONDecodeError:
                continue
    if not token:
        print("no token:", login)
        ssh.close()
        return 1
    print("login: ok\n")

    passed = failed = 0
    for ep in ENDPOINTS:
        raw = run(
            f"curl -s --max-time 120 -w '\\n__HTTP__:%{{http_code}}' "
            f"-H 'Authorization: Bearer {token}' '{base}{ep}'",
            timeout=150,
        )
        if "__HTTP__:" not in raw:
            print(f"FAIL {ep}: {raw[:120]}")
            failed += 1
            continue
        body, code = raw.rsplit("__HTTP__:", 1)
        code = code.strip()
        try:
            j = json.loads(body)
        except json.JSONDecodeError:
            print(f"FAIL {ep} http={code}: invalid json")
            failed += 1
            continue
        if j.get("code") == 0:
            data = j.get("data")
            if isinstance(data, dict):
                keys = list(data.keys())[:8]
            elif isinstance(data, list):
                keys = [f"list[{len(data)}]"]
            else:
                keys = [type(data).__name__]
            print(f"OK   {ep:32} http={code} keys={keys}")
            passed += 1
        else:
            print(f"FAIL {ep} http={code}: {j.get('error') or j}")
            failed += 1

    bundle = run(f"grep -l InfraHubView {INSTALL}/web/assets/*.js 2>/dev/null | head -1")
    print(f"\nfrontend InfraHubView bundle: {bundle or 'NOT FOUND'}")

    overview_sample = run(
        f"curl -s --max-time 120 -H 'Authorization: Bearer {token}' '{base}/infra-hub/overview'",
        timeout=150,
    )
    try:
        ov = json.loads(overview_sample).get("data") or {}
        print(
            "overview sample:",
            {
                "health_score": ov.get("health_score"),
                "llmops": bool(ov.get("llmops")),
                "dataops": bool(ov.get("dataops")),
                "aiops": bool(ov.get("aiops")),
                "secops": bool(ov.get("secops")),
                "orchestration": bool(ov.get("orchestration")),
            },
        )
    except Exception:
        pass

    ssh.close()
    print(f"\n{passed} passed, {failed} failed")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
