<p align="center">
  <img src="frontend/public/logo.svg" alt="Open Panel logo" width="128" style="border-radius: 22%;" />
</p>

<h1 align="center">Open Panel</h1>

<p align="center">
  Open-source Linux server management panel · Modern Vue 3 UI
</p>

<p align="center">
  <a href="docs/README.md"><img src="https://img.shields.io/badge/Docs-README-lightgrey?style=flat-square" alt="Docs"></a>
</p>

<p align="center">
  <img src="docs/images/dashboard.png" alt="Open Panel dashboard" width="900" />
</p>

<p align="center"><em>Dashboard preview</em></p>

---

Open Panel is an open-source Linux server management panel inspired by modern panels such as [1Panel](https://1panel.cn/), with a full web UI for day-to-day operations.

📖 **User guide:** [docs/en/USER_GUIDE.md](docs/en/USER_GUIDE.md)

### Features

| Module | Description |
|--------|-------------|
| **Dashboard** | Real-time metrics, health score, one-click optimize, global traffic map & geo policies |
| **Websites** | Nginx/OpenResty vhosts, SSL, CDN cache, WordPress toolkit, Node.js/Java runtimes |
| **Databases** | MySQL / PostgreSQL / Redis, extensions, backup/import, OSS remote backup |
| **Files / OSS** | Online file manager, archive tools, object storage sync |
| **Docker** | Container/image management, one-click Docker Compose templates |
| **FTP / Mail** | Pure-FTPd user sync, Postfix/Dovecot virtual mailboxes |
| **Backup / Cron** | Scheduled site/DB/directory backups to FTP/SFTP/WebDAV/OSS; cron templates |
| **Monitoring** | Uptime probes, cluster nodes, auto ops, DevOps center |
| **Protection** | Firewall, Nginx WAF, security scans, login audit, 2FA, optional Edge/Kafka/Cilium |
| **SSH / PAM** | Multi-tab Web SSH, AI assistant, assets/permissions/audit/JIT/account rotation/batch ops |
| **Toolbox** | Network diagnostics, ports/processes, command snippets, system health |
| **AI Hub** | AI apps and panel-level model configuration |
| **Other** | DNS, app store, extensions, phpMyAdmin, log center |
| **Users** | Admin / regular / sub-users, 9 module permissions, disk quotas |
| **i18n** | Simplified Chinese / Traditional Chinese / English |

### Tech stack

| Layer | Stack |
|-------|-------|
| Backend | Go 1.22+ / Gin / GORM / SQLite |
| Frontend | Vue 3 / Vite / Element Plus / Pinia |
| Deploy | Single binary + embedded frontend / systemd |

### Quick start

#### Requirements

- **Linux (recommended for production)**: Ubuntu 20.04+, Debian 11+, CentOS 7+, Rocky / AlmaLinux / RHEL (`apt` / `dnf` / `yum` auto-detected)
- **Windows**: Windows 10/11, Windows Server (dev/demo; some OS-level features are limited)
- **Docker**: Any host with Docker (see `docker compose` below)
- Go 1.22+ (source builds only)
- Node.js 18+ (frontend dev/build only)

#### One-click install

**Ubuntu / Debian / CentOS / Rocky / AlmaLinux**

```bash
curl -fsSL https://raw.githubusercontent.com/luuuunet/open-panel/main/scripts/install.sh | sudo bash
# Local: sudo bash scripts/install.sh
# From source: sudo FROM_SOURCE=1 bash scripts/install.sh
# Custom path: sudo INSTALL_DIR=/opt/open-panel OPEN_PANEL_PORT=8888 bash scripts/install.sh
```

**Windows (Administrator PowerShell)**

```powershell
Set-ExecutionPolicy Bypass -Scope Process -Force
cd C:\path\to\open-panel
.\scripts\install.ps1 -FromSource
# Custom: .\scripts\install.ps1 -InstallDir C:\open-panel -Port 8888
```

**Docker (Linux / Windows / macOS)**

```bash
docker compose up -d --build
# Open http://localhost:8888
# Or use the auto-setup script:
bash scripts/auto-setup.sh docker
# Windows: .\scripts\auto-setup.ps1 -Action docker
```

#### Auto-setup scripts

Unified entry points for **local build, install, remote deploy, dev mode, and Docker**.

| Script | Platform | Description |
|--------|----------|-------------|
| `scripts/auto-setup.sh` | Linux / macOS / Git Bash | Bash one-click script |
| `scripts/auto-setup.ps1` | Windows PowerShell | PowerShell one-click script |
| `scripts/deploy.py` | All platforms | Remote SSH deploy (binary + web) |
| `scripts/deploy.env.example` | — | Remote deploy config template |
| `scripts/install.sh` | Linux | Local install (used by `auto-setup install`) |
| `scripts/install.ps1` | Windows | Local install (used by `auto-setup install`) |
| `scripts/build-release.sh` | Linux | Multi-arch release packages (maintainers) |

**Dependencies**

| Mode | Requires |
|------|----------|
| `build` | Go 1.22+, Node.js 18+ |
| `install` | root on Linux; Administrator on Windows |
| `deploy` | build deps + Python 3 + `pip install paramiko` |
| `docker` | Docker + Docker Compose |

**Commands**

| Mode | Linux / Git Bash | Windows PowerShell | Purpose |
|------|------------------|--------------------|---------|
| Build | `bash scripts/auto-setup.sh build` | `.\scripts\auto-setup.ps1 -Action build` | Build frontend + cross-compile Linux binary |
| Install | `sudo bash scripts/auto-setup.sh install` | `.\scripts\auto-setup.ps1 -Action install` | Install panel locally and register service |
| Deploy | `bash scripts/auto-setup.sh deploy` | `.\scripts\auto-setup.ps1 -Action deploy` | Build, upload to remote server, restart |
| Dev | `bash scripts/auto-setup.sh dev` | `.\scripts\auto-setup.ps1 -Action dev` | Run backend (frontend: `npm run dev` in another terminal) |
| Docker | `bash scripts/auto-setup.sh docker` | `.\scripts\auto-setup.ps1 -Action docker` | `docker compose up -d --build` |

Build output goes to `dist/open-panel-linux-amd64/`:

```
dist/open-panel-linux-amd64/
├── open-panel      # Linux binary
├── op              # CLI tool
└── web/            # Frontend static assets
```

**Remote deploy configuration**

1. Copy the template:

```bash
cp scripts/deploy.env.example scripts/deploy.env
# Windows: copy scripts\deploy.env.example scripts\deploy.env
```

2. Edit `scripts/deploy.env`:

```ini
DEPLOY_HOST=your-server-ip      # Target server IP or hostname
DEPLOY_USER=root                # SSH username
DEPLOY_KEY=~/.ssh/id_rsa        # SSH private key path (recommended)
# DEPLOY_PASSWORD=              # Or use password (pick one)
OPEN_PANEL_PORT=8888            # Panel port
INSTALL_DIR=/opt/open-panel     # Remote install directory
DEPLOY_GOARCH=amd64             # Target CPU arch: amd64 | arm64
```

3. Install Python deps and deploy:

```bash
pip install paramiko

# Linux / Git Bash — build + deploy
bash scripts/auto-setup.sh deploy

# Windows PowerShell
.\scripts\auto-setup.ps1 -Action deploy
```

Deploy flow: build frontend → cross-compile Linux backend → upload binary and `web/` over SSH → restart remote `open-panel` service.

You can also use `deploy.py` directly:

```bash
# Web assets only (requires backend/web built)
python3 scripts/deploy.py --web-only

# Binary + web
python3 scripts/deploy.py --full --binary dist/open-panel-linux-amd64/open-panel
```

**Typical workflows**

```bash
# 1. Develop on Windows, deploy to Linux production
copy scripts\deploy.env.example scripts\deploy.env
# Edit deploy.env with server details
pip install paramiko
.\scripts\auto-setup.ps1 -Action deploy

# 2. Install from source on a Linux server
git clone https://github.com/luuuunet/open-panel.git
cd open-panel
sudo bash scripts/auto-setup.sh install

# 3. Build locally, copy to server manually
bash scripts/auto-setup.sh build
# Upload dist/open-panel-linux-amd64/ then:
# cp open-panel /opt/open-panel/open-panel && systemctl restart open-panel

# 4. Maintainer: multi-arch release tarballs
bash scripts/build-release.sh
# Output: dist/open-panel-linux-amd64.tar.gz, etc.
```

**Makefile shortcuts**

```bash
make setup    # same as bash scripts/auto-setup.sh build
make deploy   # same as bash scripts/auto-setup.sh deploy
make install  # same as bash scripts/install.sh
```

After install, call `GET /api/system/platform` (login required) to inspect package manager and capability flags.

#### Platform capability matrix

| Capability | Ubuntu/Debian | CentOS/RHEL family | Windows |
|------------|---------------|--------------------|---------|
| Panel web UI | ✅ | ✅ | ✅ |
| apt/dnf/yum app install | ✅ apt | ✅ dnf/yum | ⚠️ winget |
| systemd service | ✅ | ✅ | ⚠️ scheduled task |
| Firewall applied to OS | ✅ ufw/firewalld | ✅ | ❌ panel record only |
| FTP/mail sync | ✅ | ✅ | ❌ |
| Docker app store | ✅ | ✅ | ✅ (Docker Desktop) |

Windows builds run, but some OS-level features (firewall/FTP/mail sync) are simulated or skipped.

#### Source development

```bash
git clone https://github.com/luuuunet/open-panel.git
cd open-panel

# Backend
cd backend
go mod download
go run ./cmd/server

# Frontend (separate terminal, dev mode)
cd frontend
npm install
npm run dev

# Or build and embed frontend
cd frontend && npm run build
cd ../backend && go build -o open-panel ./cmd/server/
```

Default URL: `http://SERVER_IP:8888`  
First login: username `admin`, **random password** in `data/INITIAL_CREDENTIALS.txt` or server logs (`journalctl -u open-panel` / console). Change the password immediately. With security entrance enabled: `http://IP:8888/<entrance-path>/`

#### Data directory layout

All managed data lives under `dataDir` (default `/opt/open-panel/data`, configurable via `INSTALL_DIR`):

| Path | Purpose |
|------|---------|
| `{dataDir}/wwwroot` | Website document roots |
| `{dataDir}/server/{app}` | Installed apps (Nginx, MySQL, PHP, etc.) |
| `{dataDir}/apps/{app}` | One-click Docker apps |
| `{dataDir}/ai/{app}` | AI-related apps |
| `{dataDir}/logs` | Site/WAF/panel logs |
| `{dataDir}/backup` | Local backups |

#### Panel CLI `op`

```bash
op              # Interactive menu (shows panel info first)
op info         # Panel URLs, port, and data directory
op config       # Change port, entrance path, or SSL
op restart      # Restart the panel service
op help         # Full command list
```

### Sub-user permissions

Create sub-users under **User management** and assign module permissions:

`websites` · `databases` · `files` · `docker` · `ftp` · `mail` · `backup` · `monitor` · `bastion` (SSH / PAM)

API and UI enforce permissions. Software install, firewall/WAF write, DNS, logs, DevOps, AI, toolbox, and panel settings are admin-only; sub-users with `bastion` can use **SSH / PAM** for authorized assets.

See the [user guide](docs/en/USER_GUIDE.md#16-users--permissions) for details.

### Project structure

```
open-panel/
├── backend/              # Go API service
│   ├── cmd/server/       # Entry point
│   └── internal/         # Business logic
├── frontend/             # Vue 3 frontend
│   └── public/logo.svg   # Project logo
├── scripts/
│   ├── auto-setup.sh     # Auto-setup (Linux / macOS / Git Bash)
│   ├── auto-setup.ps1    # Auto-setup (Windows)
│   ├── deploy.py         # Remote SSH deploy
│   ├── deploy.env.example
│   ├── install.sh        # Linux install
│   ├── install.ps1       # Windows install
│   └── build-release.sh  # Multi-arch releases
└── docker-compose.yml
```

### Roadmap

- [x] Dashboard and system monitoring
- [x] Websites / Nginx / SSL / CDN cache
- [x] Database management and backup (incl. OSS)
- [x] File manager and object storage
- [x] Docker / Compose template deploy
- [x] Firewall / WAF / security scans
- [x] Cron and backup jobs
- [x] FTP / mail / DNS
- [x] Multi-user, sub-accounts, disk quotas
- [x] SSH / PAM (bastion merged into SSH terminal)
- [x] Traffic map geo drill-down and policies
- [x] Dashboard health score and one-click optimize
- [x] Toolbox and PostgreSQL extension management
- [x] Login 2FA and trilingual UI
- [ ] WHM-style reseller / multi-tenant
- [x] LNMP/LAMP one-click stack and readiness checks
- [ ] Backup object-storage reliability and alerts
- [ ] More Compose app-store templates
- [ ] Plugin / extension ecosystem

### Contributing

Issues and pull requests are welcome!

### License

[MIT License](LICENSE)
