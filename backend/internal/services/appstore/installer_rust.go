package appstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func tryRustInstall(key, _, installPath, dataDir string) (bool, error) {
	if !strings.HasPrefix(key, "rust") || key == "rustfs" || key == "rustdesk" {
		return false, nil
	}
	toolchain := rustToolchainFromKey(key)
	if toolchain == "" {
		return false, fmt.Errorf("unsupported Rust version key: %s", key)
	}
	return true, installRustToolchain(dataDir, toolchain, key, installPath)
}

func rustToolchainFromKey(key string) string {
	ver := strings.TrimPrefix(key, "rust")
	if len(ver) < 3 {
		return ""
	}
	// rust184 -> 1.84.0, rust183 -> 1.83.0
	major := ver[:1]
	minor := ver[1:]
	return fmt.Sprintf("%s.%s.0", major, minor)
}

func installRustToolchain(dataDir, toolchain, key, installPath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Rust %s 请通过软件商店 winget 安装 Rustlang.Rust.MSVC", toolchain)
	}
	base := installPath
	if base == "" {
		base = filepath.Join(dataDir, "server", "rust", strings.TrimPrefix(key, "rust"))
	}
	_ = os.MkdirAll(base, 0755)
	marker := filepath.Join(base, ".owpanel-installed")

	if rustc, _ := exec.LookPath("rustc"); rustc != "" {
		if ver := rustVersionString(rustc); ver != "" && strings.HasPrefix(ver, strings.TrimSuffix(toolchain, ".0")) {
			return os.WriteFile(marker, []byte(rustc+"\n"), 0644)
		}
	}

	home := os.Getenv("HOME")
	if home == "" {
		home = "/root"
	}
	cargoHome := filepath.Join(base, "cargo")
	rustupHome := filepath.Join(base, "rustup")
	env := append(os.Environ(),
		"CARGO_HOME="+cargoHome,
		"RUSTUP_HOME="+rustupHome,
		"HOME="+home,
	)

	if _, err := exec.LookPath("rustup"); err != nil {
		cmd := exec.Command("sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal --default-toolchain "+toolchain)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("安装 rustup 失败: %w (%s)", err, strings.TrimSpace(string(out)))
		}
	} else {
		cmd := exec.Command("rustup", "toolchain", "install", toolchain)
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("安装 Rust %s 失败: %w (%s)", toolchain, err, strings.TrimSpace(string(out)))
		}
	}

	rustc := filepath.Join(cargoHome, "bin", "rustc")
	if !fileExists(rustc) {
		return fmt.Errorf("Rust 安装完成但未找到 rustc")
	}
	wrapperDir := filepath.Join(base, "bin")
	_ = os.MkdirAll(wrapperDir, 0755)
	for _, name := range []string{"rustc", "cargo", "rustup"} {
		src := filepath.Join(cargoHome, "bin", name)
		if !fileExists(src) {
			continue
		}
		wrapper := filepath.Join(wrapperDir, name)
		script := fmt.Sprintf("#!/bin/bash\nexport CARGO_HOME=%q\nexport RUSTUP_HOME=%q\nexec %q \"$@\"\n", cargoHome, rustupHome, src)
		_ = os.WriteFile(wrapper, []byte(script), 0755)
	}
	return os.WriteFile(marker, []byte(rustc+"\n"), 0644)
}

func rustVersionString(rustc string) string {
	out, err := exec.Command(rustc, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	// rustc 1.84.0 (...)
	parts := strings.Fields(string(out))
	if len(parts) >= 2 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func detectRustStatus(key, dataDir string) string {
	ver := strings.TrimPrefix(key, "rust")
	marker := filepath.Join(dataDir, "server", "rust", ver, ".owpanel-installed")
	if fileExists(marker) {
		return "running"
	}
	if _, err := exec.LookPath("rustc"); err == nil {
		return "running"
	}
	return "stopped"
}

// RustBinDir returns panel-managed Rust bin dir for a version key (rust184 -> .../184/bin).
func RustBinDir(dataDir, key string) string {
	ver := strings.TrimPrefix(key, "rust")
	dir := filepath.Join(dataDir, "server", "rust", ver, "bin")
	if fileExists(filepath.Join(dir, "cargo")) {
		return dir
	}
	return ""
}
