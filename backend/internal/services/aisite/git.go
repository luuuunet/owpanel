package aisite

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveGitBinary() string {
	if p, err := exec.LookPath("git"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		for _, p := range []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Git", "cmd", "git.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "cmd", "git.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Programs", "Git", "cmd", "git.exe"),
		} {
			if p != "" && fileExists(p) {
				return p
			}
		}
	}
	return ""
}

func gitAvailable() bool {
	return resolveGitBinary() != ""
}

func shellEnvWithGit() []string {
	env := os.Environ()
	gitBin := resolveGitBinary()
	if gitBin == "" {
		return env
	}
	gitDir := filepath.Dir(gitBin)
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	merged := gitDir + string(os.PathListSeparator) + os.Getenv("PATH")
	found := false
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		prefix := pathKey + "="
		if strings.HasPrefix(e, prefix) {
			merged = gitDir + string(os.PathListSeparator) + strings.TrimPrefix(e, prefix)
			out = append(out, pathKey+"="+merged)
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		out = append(out, pathKey+"="+merged)
	}
	return out
}

func prependPath(env []string, dirs ...string) []string {
	if len(dirs) == 0 {
		return env
	}
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	prefix := strings.Join(dirs, string(os.PathListSeparator))
	found := false
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, pathKey+"=") {
			merged := prefix + string(os.PathListSeparator) + strings.TrimPrefix(e, pathKey+"=")
			out = append(out, pathKey+"="+merged)
			found = true
			continue
		}
		out = append(out, e)
	}
	if !found {
		out = append(out, pathKey+"="+prefix+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	return out
}

func ensureGitAvailable(appendLog func(string)) error {
	if gitAvailable() {
		appendLog("Git 已就绪")
		return nil
	}
	appendLog("未检测到 Git，正在自动安装（GitHub 部署必需，可能需要数分钟）…")
	switch runtime.GOOS {
	case "linux":
		script := "export DEBIAN_FRONTEND=noninteractive; " +
			"(command -v apt-get >/dev/null && apt-get update -qq && apt-get install -y git) || " +
			"(command -v dnf >/dev/null && dnf install -y git) || " +
			"(command -v yum >/dev/null && yum install -y git) || " +
			"(command -v apk >/dev/null && apk add --no-cache git)"
		cmd := exec.Command("bash", "-c", script)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Linux 自动安装 Git 失败: %w (%s)", err, strings.TrimSpace(string(out)))
		}
	case "windows":
		if err := installGitWindows(appendLog); err != nil {
			return err
		}
	default:
		return fmt.Errorf("当前系统未安装 Git，请先手动安装后再部署 GitHub 项目")
	}
	if !gitAvailable() {
		return fmt.Errorf("Git 安装完成但仍不可用，请重启面板服务或手动安装 Git 后重试")
	}
	appendLog("Git 安装完成")
	return nil
}

func installGitWindows(appendLog func(string)) error {
	if _, err := exec.LookPath("winget"); err == nil {
		appendLog("使用 winget 安装 Git…")
		cmd := exec.Command("winget", "install", "--id", "Git.Git", "-e",
			"--accept-source-agreements", "--accept-package-agreements", "--silent")
		out, err := cmd.CombinedOutput()
		text := strings.TrimSpace(string(out))
		if err != nil && !strings.Contains(strings.ToLower(text), "already installed") {
			appendLog("winget 安装 Git: " + text)
		}
		if gitAvailable() {
			return nil
		}
	}
	if _, err := exec.LookPath("choco"); err == nil {
		appendLog("使用 Chocolatey 安装 Git…")
		cmd := exec.Command("choco", "install", "git", "-y")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("choco 安装 Git 失败: %w (%s)", err, strings.TrimSpace(string(out)))
		}
		if gitAvailable() {
			return nil
		}
	}
	if resolveGitBinary() != "" {
		return nil
	}
	return fmt.Errorf("请手动安装 Git（https://git-scm.com/download/win）或确保 winget/choco 可用")
}
