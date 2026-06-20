# Rust Runtime

OWPanel lets you install the Rust toolchain from the **App Store** and manage Rust web projects (Actix Web, Axum, Rocket, etc.) on the **Runtimes** page.

## Quick start

### 1. Install Rust

**App Store → Runtime → Rust**

- Pick a version (e.g. **1.84**, **1.83**) and install
- The panel uses `rustup` to install `rustc` / `cargo`; multiple versions can coexist
- Also install **PM2** (process manager) and optionally **Docker** (container mode)

### 2. Create a runtime project

**Websites → Runtimes → Rust** tab

| Field | Description |
|-------|-------------|
| Name | Project id, e.g. `my-api` |
| Code directory | Project root containing `Cargo.toml` |
| Version | Toolchain version (1.84 / 1.83 / 1.82) |
| Run script | Compiled binary, e.g. `./target/release/my-api` |
| External port | Service port; Nginx will proxy here |

**Execution modes:**

- **Compiled binary + PM2** (recommended): run `cargo build --release`, then set run script to `./target/release/<crate-name>`
- **Docker**: with Docker installed, the panel uses `rust:<version>` images; run script can be `cargo build --release && ./target/release/app`

### 3. Bind a website (Nginx reverse proxy)

1. Create a site under **Websites** (PHP/static is fine)
2. Set **Reverse proxy** to `http://127.0.0.1:<port>`
3. Or link the project after starting the runtime

## Example run scripts

```bash
# After cargo build --release on the host
./target/release/my-api

# Build and run inside Docker
cargo build --release && ./target/release/my-api

# With PORT env (most frameworks honor PORT)
PORT=8080 ./target/release/my-api
```

Set `PORT=8080` under **Environment variables** in the runtime form; the panel injects it on start.

## AI / GitHub deploy

When importing a repo with `Cargo.toml` via **AI site** or **DevOps**, the panel will:

1. Detect a **Rust** project
2. Install Rust toolchain and PM2
3. Run `cargo build --release`
4. Create a runtime and configure Nginx proxy

## Framework notes

| Framework | Typical run script |
|-----------|------------------|
| Axum / Actix Web | `./target/release/<crate-name>` |
| Rocket | `./target/release/<crate-name>` |
| Custom `[[bin]]` | See `name` under `[bin]` in `Cargo.toml` |

The default binary name comes from `[package] name` in `Cargo.toml`; override the run script if your binary name differs.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| PM2 fails to start | Ensure `cargo build --release` succeeded and the path in run script is correct |
| Slow Docker start | First run compiles inside the container; pre-build on host and use PM2 instead |
| 502 Bad Gateway | Verify service port matches Nginx proxy URL |
| cargo not found | Reinstall the Rust version from the App Store |

## See also

- [User Guide — Runtimes](./USER_GUIDE.md)
- [User Guide — App Store](./USER_GUIDE.md)
