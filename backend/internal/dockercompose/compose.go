package dockercompose

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	composeV1NotFound = "is not a docker command"
)

// HasV2 reports whether `docker compose` (Compose plugin) is available and functional.
func HasV2() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "compose", "version").CombinedOutput()
	if err != nil {
		return false
	}
	s := strings.ToLower(string(out))
	return strings.Contains(s, "compose version") || strings.Contains(s, "docker compose")
}

func composeV1Path() string {
	candidates := []string{"docker-compose", "/usr/bin/docker-compose", "/usr/local/bin/docker-compose"}
	for _, c := range candidates {
		if path, err := exec.LookPath(c); err == nil {
			return path
		}
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

// Argv returns the executable and args prefix for Compose v2 (docker compose) or v1 (docker-compose).
func Argv(extra ...string) (string, []string, error) {
	if HasV2() {
		return "docker", append([]string{"compose"}, extra...), nil
	}
	if path := composeV1Path(); path != "" {
		return path, extra, nil
	}
	return "", nil, fmt.Errorf("未找到 docker compose（请安装 docker-compose-plugin 或 docker-compose）")
}

func defaultComposeInDir(dir, composeFile string) bool {
	if composeFile == "" {
		return true
	}
	dir = filepath.Clean(dir)
	composeFile = filepath.Clean(composeFile)
	base := filepath.Base(composeFile)
	switch base {
	case "docker-compose.yml", "docker-compose.yaml", "compose.yml":
	default:
		return false
	}
	return composeFile == filepath.Join(dir, base)
}

func composeArgs(dir, composeFile string, args ...string) []string {
	if composeFile == "" || defaultComposeInDir(dir, composeFile) {
		return args
	}
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(composeFile))
	if err == nil && !strings.HasPrefix(rel, "..") {
		return append([]string{"-f", rel}, args...)
	}
	return append([]string{"-f", composeFile}, args...)
}

func isComposeMissing(err error, text string) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(text)
	if strings.Contains(msg, composeV1NotFound) {
		return true
	}
	if strings.Contains(msg, "unknown shorthand flag: 'f' in -f") {
		return true
	}
	if strings.Contains(msg, "no such file or directory") && strings.Contains(msg, "docker-compose") {
		return true
	}
	return false
}

func runOnce(dir, name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("%s", text)
	}
	return text, nil
}

// RunInDir executes compose in dir with optional explicit compose file (-f).
func RunInDir(dir, composeFile string, args ...string) (string, error) {
	args = composeArgs(dir, composeFile, args...)
	if name, cmdArgs, err := Argv(args...); err == nil {
		text, runErr := runOnce(dir, name, cmdArgs)
		if runErr == nil || !isComposeMissing(runErr, text) {
			return text, runErr
		}
	}
	// Fallback: if v2 was chosen but failed, retry docker-compose v1.
	if path := composeV1Path(); path != "" && !HasV2() {
		return runOnce(dir, path, args)
	}
	if path := composeV1Path(); path != "" {
		if text, err := runOnce(dir, path, args); err == nil {
			return text, nil
		} else if !isComposeMissing(err, text) {
			return text, err
		}
	}
	name, cmdArgs, err := Argv(args...)
	if err != nil {
		return "", err
	}
	return runOnce(dir, name, cmdArgs)
}

// PSRunning reports whether any service is running in dir (default compose file).
func PSRunning(dir string) bool {
	name, cmdArgs, err := Argv("ps", "--status", "running", "-q")
	if err != nil {
		return false
	}
	text, runErr := runOnce(dir, name, cmdArgs)
	if runErr != nil && isComposeMissing(runErr, text) {
		if path := composeV1Path(); path != "" {
			text, runErr = runOnce(dir, path, cmdArgs)
		}
	}
	if runErr != nil {
		return false
	}
	return strings.TrimSpace(text) != ""
}
