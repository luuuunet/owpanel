//go:build !windows

package appstore

import (
	"os/exec"
	"strings"
)

func detectWindowsServiceStatus(_, _ string) string {
	return "stopped"
}

func processExists(name string) bool {
	out, err := exec.Command("pgrep", "-x", name).CombinedOutput()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}
