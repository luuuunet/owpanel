# OWPanel Automation Guide (Beginner-Friendly)

> Plain-language guide to the **Automation** menu: what each feature does, how it maps to aaPanel, 1Panel, Alibaba Cloud, AWS, and Google Cloud, and how to get started.

---

## 1. What problem does automation solve?

| Pain | OWPanel helps with |
|------|-------------------|
| Site down overnight | Service watch + auto-restart, uptime probes |
| Forgot to backup | Scheduled backups (like cloud snapshots) |
| SSL expired | Auto-renew Let's Encrypt |
| CPU/RAM full | Resource alerts + webhook to phone |
| SSH to run scripts | Cron jobs |

All of this lives under **Automation** in the sidebar — **all-in-one on a single server**.

---

## 2. Migration cheat sheet

| Goal | OWPanel | aaPanel / 1Panel | Alibaba SAS | AWS Lightsail | Google Cloud |
|------|---------|------------------|-------------|---------------|--------------|
| CPU/RAM/disk | Auto Ops → Overview | Monitoring | Instance metrics | Metrics | Cloud Monitoring |
| Auto-restart services | Auto Ops → Watch list | Service guard | — | — | — |
| HTTPS + renew | SSL | SSL | One-click HTTPS | LB certs | Managed certs |
| Scheduled backup | Backup | Backup / cron | Manual snapshot | Auto snapshots | Snapshot schedules |
| External HTTP check | Uptime | Reports | Basic | Metric alarms | Uptime checks |
| Scheduled scripts | Cron | Cron | Command assistant | CLI/API | Cloud Scheduler |
| Alerts | Auto Ops → Webhook | Notifications | Email | Email/SMS | Alert policies |

---

## 3. Recommended: one-click site protection

**Automation → Auto Operations → Getting started**

**Enable site protection** automatically:

1. Enables auto ops, SSL renew, website health scans  
2. Watch + auto-restart for Nginx/MySQL/PHP etc.  
3. Uptime probes for all running sites (5 min interval)  
4. Daily 2:00 backup tasks for all websites  
5. Daily 3:00 backup tasks for all databases  

Then set a **Webhook** under **Policy** for DingTalk, Slack, Discord, etc.

---

## 4. Feature details

### Auto Operations (hub)

| Tab | Purpose |
|-----|---------|
| Getting started | Comparison table, paths, presets |
| Overview | CPU/memory/disk gauges |
| Watch list | Enable watch / auto-restart per app |
| Events | Restart and alert history |
| Policy | Intervals, webhook, thresholds |
| Website audit | Health score and recommendations |

### Uptime

Probes URLs like a visitor. **Import from websites** creates monitors automatically.

### Backup

**Backup all websites** / **Backup all databases** quick templates. Remote: FTP, SFTP, WebDAV, OSS.

### Cron

Built-in templates: Docker prune, nginx reload, SSL renew, log cleanup, etc.

---

## 5. Glossary

- **Webhook** — HTTPS URL; panel POSTs JSON on incidents  
- **Cron** — Schedule syntax, e.g. `0 2 * * *` = daily at 2 AM  
- **Snapshot / backup** — Point-in-time copy for restore  
- **Uptime probe** — HTTP check from outside, closer to user experience  
- **Watch / auto-restart** — Restarts stopped services automatically  

---

## 6. OWPanel extras (rare on cloud consoles)

- Website health audit (SEO, headers, broken links)  
- A/B testing & product analytics  
- DevOps CI/CD & CVE scan  
- Memory auto-relief  
- Extension event hooks  

---

## 7. FAQ

**Will one-click protection overwrite backups?**  
No — existing tasks for the same site/DB are skipped.

**What vs cloud monitoring?**  
No global multi-region probes or built-in SMS; use webhooks. For large fleets, add Prometheus/Grafana.

**Cron vs Backup?**  
Backup is for site/DB packages; Cron runs any shell command.

---

See also [USER_GUIDE.md](./USER_GUIDE.md) §9.
