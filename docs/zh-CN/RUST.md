# Rust 运行环境

OWPanel 支持在**软件商店**安装 Rust 工具链，并在**运行环境**页面管理 Rust Web 项目（Actix Web、Axum、Rocket 等）。

## 快速开始

### 1. 安装 Rust

路径：**软件商店 → 运行环境 → Rust**

- 选择版本（如 **1.84**、**1.83**）点击安装
- 面板通过 `rustup` 安装 `rustc` / `cargo`，多版本可并存
- 建议同时安装 **PM2**（进程守护）和 **Docker**（可选，容器方式运行）

### 2. 创建运行环境项目

路径：**网站 → 运行环境 → Rust** 标签

| 字段 | 说明 |
|------|------|
| 名称 | 项目标识，如 `my-api` |
| 代码目录 | 含 `Cargo.toml` 的项目根目录 |
| 版本 | Rust 工具链版本（1.84 / 1.83 / 1.82） |
| 启动命令 | 已编译二进制，如 `./target/release/my-api` |
| 外部端口 | 服务监听端口，Nginx 将反代到此端口 |

**运行方式：**

- **已编译二进制 + PM2**（推荐）：先 `cargo build --release`，启动命令填 `./target/release/<包名>`
- **Docker 容器**：安装 Docker 后，面板用 `rust:<版本>` 镜像，启动命令可填 `cargo build --release && ./target/release/app`

### 3. 绑定网站（Nginx 反代）

1. 在 **网站** 中创建站点，PHP 版本选「静态」或任意
2. 编辑站点，设置 **反向代理** 为 `http://127.0.0.1:<端口>`
3. 或在运行环境启动后，从网站列表关联该项目

## 常用启动命令示例

```bash
# 本地已 cargo build --release 后
./target/release/my-api

# 容器内编译并运行（Docker 模式）
cargo build --release && ./target/release/my-api

# 指定端口（多数框架读取 PORT 环境变量）
PORT=8080 ./target/release/my-api
```

在运行环境 **环境变量** 中可设置 `PORT=8080`，面板启动时会自动注入。

## AI / GitHub 一键部署

从 **AI 建站** 或 **DevOps** 导入含 `Cargo.toml` 的仓库时，面板会：

1. 识别为 **Rust** 项目
2. 自动安装 Rust 工具链与 PM2
3. 执行 `cargo build --release`
4. 创建运行环境并配置 Nginx 反代

## 框架提示

| 框架 | 典型启动方式 |
|------|-------------|
| Axum / Actix Web | `./target/release/<crate-name>` |
| Rocket | `./target/release/<crate-name>` |
| 自定义 [[bin]] | 见 `Cargo.toml` 中 `[bin]` 的 `name` |

包名默认从 `Cargo.toml` 的 `[package] name` 读取；若与二进制名不同，请在启动命令中手动指定路径。

## 故障排查

| 现象 | 处理 |
|------|------|
| PM2 启动失败 | 确认已 `cargo build --release`，且启动命令路径正确 |
| Docker 启动慢 | 首次需在容器内编译，可先在主机编译再用 PM2 |
| 502 Bad Gateway | 检查服务端口与 Nginx 反代地址是否一致 |
| 找不到 cargo | 在软件商店重新安装对应 Rust 版本 |

## 相关文档

- [用户手册 - 运行环境](./USER_GUIDE.md#43-运行环境)
- [用户手册 - 软件商店](./USER_GUIDE.md#14-软件商店与扩展)
