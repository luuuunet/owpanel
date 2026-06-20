#!/usr/bin/env python3
"""Automated API smoke tests for OWPanel — runs HTTP via SSH on the target server."""
from __future__ import annotations

import json
import os
import re
import sys
import time
from dataclasses import dataclass, field
from pathlib import Path

import paramiko

ROOT = Path(__file__).parent.parent
ENV_FILE = Path(__file__).parent / "deploy.env"
RESET_BIN = ROOT / "dist" / "reset-admin-linux-amd64"
TEST_PWD = os.environ.get("SMOKE_ADMIN_PASSWORD", "SmokeTest123!")


@dataclass
class Result:
    path: str
    ok: bool
    code: int | None = None
    detail: str = ""


@dataclass
class Report:
    passed: list[Result] = field(default_factory=list)
    failed: list[Result] = field(default_factory=list)
    skipped: list[Result] = field(default_factory=list)


def load_cfg() -> dict[str, str]:
    cfg: dict[str, str] = {}
    if ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            cfg[k.strip()] = v.strip()
    for k in ("DEPLOY_HOST", "DEPLOY_USER", "DEPLOY_PASSWORD", "OWPANEL_PORT", "INSTALL_DIR"):
        if os.environ.get(k):
            cfg[k] = os.environ[k]
    return cfg


# GET endpoints without path params (admin user has all permissions)
# (need_auth, path) — health is outside /api/v1
SMOKE_GETS = [
    # public (health uses prefix root, not api/v1)
    (False, "__ROOT__/health"),
    (False, "/auth/bootstrap"),
    # auth
    (True, "/auth/me"),
    # dashboard
    (True, "/dashboard/stats"),
    (True, "/dashboard/processes"),
    (True, "/dashboard/history"),
    (True, "/dashboard/monitor?lite=1"),
    (True, "/dashboard/alerts"),
    (True, "/dashboard/health"),
    (True, "/dashboard/performance"),
    # system
    (True, "/system/platform"),
    (True, "/system/readiness"),
    (True, "/system/stacks"),
    # websites
    (True, "/websites"),
    (True, "/websites/projects"),
    (True, "/websites/webserver"),
    (True, "/websites/options"),
    (True, "/ai/assistant/status"),
    (True, "/websites/ai/jobs"),
    # ssl/cache/analytics
    (True, "/ssl"),
    (True, "/ssl/status"),
    (True, "/cache/config"),
    (True, "/cache/status"),
    (True, "/cache/sites"),
    (True, "/cache/rules"),
    (True, "/analytics/traffic-map"),
    (True, "/analytics/geo-policies"),
    (True, "/product-analytics/status"),
    # wordpress/nodejs/java
    (True, "/wordpress"),
    (True, "/nodejs"),
    (True, "/java"),
    # runtime
    (True, "/runtimes"),
    (True, "/runtimes/versions"),
    # databases
    (True, "/databases"),
    (True, "/databases/mysql/status"),
    (True, "/databases/mongodb/status"),
    (True, "/databases/pgsql/status"),
    (True, "/databases/engines/status"),
    (True, "/databases/pgsql/extensions"),
    # files / oss
    (True, "/files/roots"),
    (True, "/files/trash"),
    (True, "/oss/providers"),
    (True, "/oss/storages"),
    (True, "/oss/sync-tasks"),
    (True, "/oss/export"),
    # docker
    (True, "/docker/status"),
    (True, "/docker/containers"),
    (True, "/docker/images"),
    (True, "/docker/volumes"),
    (True, "/docker/networks"),
    (True, "/compose/templates"),
    (True, "/compose"),
    # ftp/mail/backup/cron
    (True, "/ftp"),
    (True, "/mail/status"),
    (True, "/mail/domains"),
    (True, "/mail/mailboxes"),
    (True, "/mail/webmail"),
    (True, "/mail/bulk/providers/catalog"),
    (True, "/mail/bulk/providers"),
    (True, "/mail/bulk/campaigns"),
    (True, "/backup"),
    (True, "/backup/remotes"),
    (True, "/cron"),
    (True, "/cron/templates"),
    (True, "/cron/status"),
    # monitor
    (True, "/uptime"),
    (True, "/auto-ops/status"),
    (True, "/auto-ops/overview"),
    (True, "/auto-ops/events"),
    (True, "/auto-ops/website-audits"),
    (True, "/cloud/hub"),
    (True, "/cluster/overview"),
    (True, "/cluster/nodes"),
    (True, "/cluster/join-info"),
    (True, "/cluster/workflow"),
    (True, "/load-balancers"),
    # admin
    (True, "/firewall"),
    (True, "/firewall/status"),
    (True, "/apps"),
    (True, "/software/store"),
    (True, "/software/installed"),
    (True, "/waf"),
    (True, "/waf/config"),
    (True, "/waf/status"),
    (True, "/waf/blacklist"),
    (True, "/waf/whitelist"),
    (True, "/waf/preview"),
    (True, "/waf/geoip/countries"),
    (True, "/waf/geoip/status"),
    (True, "/waf/crawlers"),
    (True, "/waf/crawlers/rules"),
    (True, "/dns"),
    (True, "/dns/providers"),
    (True, "/dns/providers/supported"),
    (True, "/dns/zones"),
    (True, "/dns/detect"),
    (True, "/dns/server-ip"),
    (True, "/logs"),
    (True, "/logs/sources"),
    (True, "/logs/retention"),
    (True, "/security/scan"),
    (True, "/security/login-logs"),
    (True, "/kafka-accel/status"),
    (True, "/cilium/dashboard"),
    (True, "/cilium/status"),
    (True, "/cilium/config"),
    (True, "/cilium/policies"),
    (True, "/cilium/presets"),
    (True, "/k8s/dashboard"),
    (True, "/k8s/status"),
    (True, "/k8s/join-info"),
    (True, "/k8s/nodes"),
    (True, "/k8s/pods"),
    (True, "/k8s/deployments"),
    (True, "/k8s/namespaces"),
    (True, "/k8s/settings"),
    (True, "/devops/deploy/configs"),
    (True, "/devops/deploy/jobs"),
    (True, "/devops/diagnostics/slow-logs"),
    (True, "/devops/diagnostics/traffic-anomalies"),
    (True, "/devops/audit/config"),
    (True, "/devops/security/cve"),
    (True, "/ai/hub/status"),
    (True, "/ai/gpu"),
    (True, "/ai/agents"),
    (True, "/ai/huggingface/status"),
    (True, "/ai/huggingface/catalog"),
    (True, "/ai/huggingface/tasks"),
    (True, "/ai/huggingface/token"),
    (True, "/toolbox/health"),
    (True, "/toolbox/system/overview"),
    (True, "/toolbox/system/ports"),
    (True, "/toolbox/system/processes"),
    (True, "/toolbox/snippets"),
    (True, "/terminal/targets"),
    (True, "/terminal/keys"),
    (True, "/users"),
    (True, "/settings"),
    (True, "/settings/migration/preview"),
    (True, "/enterprise/overview"),
    (True, "/enterprise/ha"),
    (True, "/enterprise/monitoring"),
    (True, "/enterprise/compliance"),
    (True, "/enterprise/audit-logs"),
    (True, "/enterprise/audit-settings"),
    (True, "/edge-workers"),
    (True, "/edge-workers/available-domains"),
    (True, "/edge-workers/preview"),
    (True, "/edge-workers/templates"),
    (True, "/edge-workers/runtime"),
    (True, "/edge-workers/kv/namespaces"),
    (True, "/edge-workers/d1/databases"),
    (True, "/extensions/menu"),
    (True, "/bastion/assets"),
    (True, "/bastion/connect-targets"),
    (True, "/bastion/accounts"),
    (True, "/bastion/sessions"),
    (True, "/bastion/groups"),
    (True, "/bastion/permissions"),
    (True, "/bastion/command-policy"),
    (True, "/bastion/command-audits"),
    (True, "/bastion/active-sessions"),
    (True, "/bastion/ops/templates"),
    (True, "/bastion/ops/jobs"),
    (True, "/bastion/ops/adhoc/history"),
    (True, "/php/versions"),
    (True, "/nginx/status"),
]

# Software keys to probe GET /software/:key
SOFTWARE_KEYS = [
    "nginx", "docker", "dotnet10", "dotnet8", "minio", "portainer",
    "redis", "mysql", "mariadb", "nodejs20", "python312", "rust184",
]


class Remote:
    def __init__(self, cfg: dict[str, str]):
        self.host = cfg["DEPLOY_HOST"]
        self.user = cfg.get("DEPLOY_USER", "root")
        self.pwd = cfg.get("DEPLOY_PASSWORD", "")
        self.port = int(cfg.get("OWPANEL_PORT", "8888"))
        self.install = cfg.get("INSTALL_DIR", "/opt/owpanel")
        self.client = paramiko.SSHClient()
        self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())
        self.client.connect(
            self.host, username=self.user, password=self.pwd, timeout=30, banner_timeout=60
        )

    def run(self, cmd: str, timeout: int = 120) -> tuple[str, str]:
        _, o, e = self.client.exec_command(cmd, timeout=timeout)
        return o.read().decode("utf-8", "replace").strip(), e.read().decode("utf-8", "replace").strip()

    def close(self):
        self.client.close()


def discover_prefix(r: Remote) -> str:
    out, _ = r.run(f"{r.install}/op info 2>/dev/null || true")
    for line in out.splitlines():
        m = re.search(rf":{r.port}/([^/\s]+)", line)
        if m:
            return m.group(1)
    out, _ = r.run(
        f"sqlite3 {r.install}/data/panel.db \"SELECT value FROM panel_settings WHERE key='panel_safe_path' LIMIT 1\""
    )
    return out.strip().strip("/") or "login"


def ensure_admin(r: Remote) -> None:
    if RESET_BIN.exists():
        sftp = r.client.open_sftp()
        sftp.put(str(RESET_BIN), "/tmp/reset-admin")
        sftp.chmod("/tmp/reset-admin", 0o755)
        sftp.close()
        r.run(f"OWPANEL_DATA={r.install}/data /tmp/reset-admin '{TEST_PWD}'", timeout=30)
    else:
        r.run(
            f"cd {r.install}/backend 2>/dev/null; "
            f"GOOS=linux GOARCH=amd64 go build -o /tmp/reset-admin ./cmd/reset-admin 2>/dev/null && "
            f"OWPANEL_DATA={r.install}/data /tmp/reset-admin '{TEST_PWD}' || true",
            timeout=180,
        )


def api_call(r: Remote, prefix: str, method: str, path: str, token: str | None = None, body: dict | None = None) -> tuple[int, dict | str]:
    base = f"http://127.0.0.1:{r.port}/{prefix}/api/v1{path}"
    headers = ["-H 'Content-Type: application/json'"]
    if token:
        headers.append(f"-H 'Authorization: Bearer {token}'")
    if body is not None:
        payload = json.dumps(body).replace("'", "'\\''")
        cmd = f"curl -s -w '\\n__HTTP__:%{{http_code}}' -X {method} {' '.join(headers)} -d '{payload}' '{base}'"
    else:
        cmd = f"curl -s -w '\\n__HTTP__:%{{http_code}}' -X {method} {' '.join(headers)} '{base}'"
    out, err = r.run(cmd, timeout=60)
    if "__HTTP__:" not in out:
        return 0, err or out or "empty"
    body_text, http = out.rsplit("__HTTP__:", 1)
    try:
        return int(http.strip()), json.loads(body_text) if body_text.strip() else {}
    except json.JSONDecodeError:
        return int(http.strip()), body_text.strip()


def ok_response(code: int, data: dict | str, need_auth: bool) -> tuple[bool, str]:
    if isinstance(data, str):
        return False, data[:200]
    api_code = data.get("code")
    if api_code == 0:
        return True, "ok"
    if api_code in (404,) and need_auth:
        return True, "not found (endpoint ok)"
    err = data.get("error") or data.get("message") or str(data)[:200]
    if code == 404:
        return True, "404"
    if code == 403 and not need_auth:
        return True, "403 expected"
    return False, f"http={code} api={api_code} {err}"


def run_smoke(cfg: dict[str, str]) -> Report:
    report = Report()
    r = Remote(cfg)
    try:
        # infra checks
        svc, _ = r.run("systemctl is-active owpanel")
        if svc != "active":
            report.failed.append(Result("systemctl owpanel", False, detail=svc or "inactive"))
            return report
        report.passed.append(Result("systemctl owpanel", True))

        prefix = discover_prefix(r)
        ensure_admin(r)

        http, boot = api_call(r, prefix, "GET", "/auth/bootstrap")
        ok, detail = ok_response(http, boot, False)
        (report.passed if ok else report.failed).append(Result("GET /auth/bootstrap", ok, http, detail))

        http, login = api_call(
            r, prefix, "POST", "/auth/login",
            body={"username": "admin", "password": TEST_PWD},
        )
        if not isinstance(login, dict) or not login.get("data", {}).get("token"):
            report.failed.append(Result("POST /auth/login", False, http, str(login)[:200]))
            return report
        token = login["data"]["token"]
        report.passed.append(Result("POST /auth/login", True))

        for need_auth, path in SMOKE_GETS:
            if path.startswith("__ROOT__/"):
                root_path = path.replace("__ROOT__/", "")
                cmd_path = f"http://127.0.0.1:{r.port}/{prefix}/{root_path}"
                out, _ = r.run(
                    f"curl -s -w '\\n__HTTP__:%{{http_code}}' '{cmd_path}'"
                )
                if "__HTTP__:" not in out:
                    report.failed.append(Result(f"GET /{root_path}", False, detail=out[:120]))
                    continue
                body_text, http = out.rsplit("__HTTP__:", 1)
                try:
                    data = json.loads(body_text) if body_text.strip() else {}
                except json.JSONDecodeError:
                    data = body_text
                ok, detail = ok_response(int(http.strip()), data, False)
                (report.passed if ok else report.failed).append(
                    Result(f"GET /{root_path}", ok, int(http.strip()), detail)
                )
                continue
            http, data = api_call(r, prefix, "GET", path, token=token if need_auth else None)
            ok, detail = ok_response(http, data, need_auth)
            entry = Result(f"GET {path}", ok, http, detail)
            (report.passed if ok else report.failed).append(entry)
            time.sleep(0.05)

        for key in SOFTWARE_KEYS:
            path = f"/software/{key}"
            http, data = api_call(r, prefix, "GET", path, token=token)
            ok, detail = ok_response(http, data, True)
            entry = Result(f"GET {path}", ok, http, detail)
            (report.passed if ok else report.failed).append(entry)

        # store install API sanity (no-op if already installed)
        for key, ver in [("portainer", "latest"), ("dotnet10", "10.0")]:
            http, data = api_call(
                r, prefix, "POST", f"/software/{key}/install",
                token=token, body={"version": ver},
            )
            if isinstance(data, dict):
                api_code = data.get("code", -1)
                err = (data.get("error") or "").lower()
                ok = api_code == 0 or "already installed" in err or "in progress" in err
                detail = data.get("error") or data.get("message") or "started"
            else:
                ok = False
                detail = str(data)[:120]
            entry = Result(f"POST /software/{key}/install", ok, http, detail)
            (report.passed if ok else report.failed).append(entry)

    finally:
        r.close()
    return report


def main() -> int:
    cfg = load_cfg()
    if not cfg.get("DEPLOY_HOST"):
        print("ERROR: set DEPLOY_HOST in deploy.env or environment")
        return 2

    print(f"[smoke] target {cfg['DEPLOY_HOST']}:{cfg.get('OWPANEL_PORT', '8888')}")
    report = run_smoke(cfg)

    print(f"\n=== PASSED {len(report.passed)} ===")
    print(f"=== FAILED {len(report.failed)} ===")
    for f in report.failed:
        print(f"  FAIL  {f.path}  ({f.detail})")

    out = ROOT / "scripts" / "smoke-report.json"
    out.write_text(
        json.dumps(
            {
                "passed": len(report.passed),
                "failed": len(report.failed),
                "failures": [{"path": x.path, "detail": x.detail} for x in report.failed],
            },
            indent=2,
            ensure_ascii=False,
        ),
        encoding="utf-8",
    )
    print(f"\nReport: {out}")
    return 1 if report.failed else 0


if __name__ == "__main__":
    sys.exit(main())
