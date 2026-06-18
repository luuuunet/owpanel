package database

import (
	"os"
	"os/exec"
	"strings"
)

const mysqlPasswordWarning = "Using a password on the command line interface can be insecure"

func cleanMySQLCLIOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "mysql: [Warning]") {
			continue
		}
		if strings.Contains(line, mysqlPasswordWarning) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func isValidMySQLSyncName(name string) bool {
	name = strings.TrimSpace(strings.Trim(name, "'\""))
	if name == "" || len(name) > 64 {
		return false
	}
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "mysql:") || strings.Contains(lower, "warning") {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
}

func mysqlExecWithPassword(bin string, baseArgs []string, password string) ([]byte, error) {
	cmd := exec.Command(bin, baseArgs...)
	if password != "" {
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+password)
	}
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, err
		}
		return nil, err
	}
	return []byte(cleanMySQLCLIOutput(string(out))), nil
}

func extractMySQLUserFromOutput(raw string) string {
	raw = cleanMySQLCLIOutput(raw)
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.Trim(line, "'\""))
		if isValidMySQLSyncName(line) {
			return line
		}
	}
	return ""
}
