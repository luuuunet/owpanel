<h1 align="center">OWPanel</h1>

<p align="center">
  <strong>开源自托管 · 去中心化 · 自动化 Linux 服务器管理面板</strong>
</p>

<p align="center">
  <a href="https://github.com/luuuunet/owpanel/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://github.com/luuuunet/owpanel"><img src="https://img.shields.io/badge/language-Go-green.svg" alt="Language"></a>
  <a href="https://github.com/luuuunet/owpanel"><img src="https://img.shields.io/badge/frontend-Vue3-brightgreen.svg" alt="Frontend"></a>
</p>

<p align="center">
  <a href="#-快速安装">快速安装</a> ·
  <a href="docs/README.md">文档中心</a> ·
  <a href="https://github.com/luuuunet/owpanel">GitHub 仓库</a>
</p>

---

**OWPanel** 是面向 Linux 服务器的开源自托管运维面板。我们坚持“数据本地化”原则，不绑定任何云端厂商账号；通过 Web 界面统一管理网站、数据库、Docker、安全与自动化运维。

## 🚀 核心亮点

* **🛡️ 完全自托管/去中心化**：单二进制部署，彻底摆脱第三方云平台依赖。
* **⚡ 轻量化架构**：Go 编写后端，预编译包仅约 **16 MB**，在 1 GB 内存的 VPS 上即可从容运行。
* **🛠️ 全栈运维与 DevOps**：从基础环境到 K8s 集群，从邮件服务到安全审计，提供企业级管控能力。
* **🤖 智能化辅助**：集成 AI 助手，支持日志自动分析、终端辅助及自动化运维工作流。
* **🔐 高安全性**：内置 PAM 堡垒机、WAF 防火墙及多重安全策略，确保数据与访问安全。

## 🛠 功能生态

| 分类 | 核心能力 |
| :--- | :--- |
| **⚙️ 自动化运维** | 计划任务、备份系统、可用性监控、DevOps 中心 |
| **🌐 集群与编排** | 服务器集群代理、Kubernetes 集群管理 |
| **🛡️ 安全与访问** | PAM 堡垒机、防火墙、WAF、CDN 缓存、Fail2ban |
| **📧 邮件与传输** | Postfix/Dovecot 一键部署、FTP 管理、DNS 解析对接 |
| **📊 智能运维** | 仪表盘、健康评分、全栈日志聚合、AI 辅助诊断 |

## 📊 可视化管理

### 智能 AI 运维中心
集成全栈日志聚合功能，利用 AI 自动诊断系统与应用错误，并提供修复建议。
![Log Center](docs/images/log-center-ai.png)

### 终端体验
内置高性能 Web SSH 终端，支持多标签页、AI 辅助与密钥管理。
![Dashboard](https://github.com/luuuunet/owpanel/raw/main/docs/images/ss1.png)

## ⚡ 快速安装

推荐使用单行命令进行快速部署（约 1–2 分钟完成）：

```bash
curl -fsSL [https://raw.githubusercontent.com/luuuunet/owpanel/v0.1.15/scripts/install.sh](https://raw.githubusercontent.com/luuuunet/owpanel/v0.1.15/scripts/install.sh) | sudo bash

### Documentation

- [English User Guide](docs/en/USER_GUIDE.md)
- [中文用户手册](docs/zh-CN/USER_GUIDE.md)
- [存储生命周期与云备份](docs/zh-CN/LIFECYCLE.md) · [Storage Lifecycle (EN)](docs/en/LIFECYCLE.md)
- [Docs index](docs/README.md)

### License

[MIT License](LICENSE)
