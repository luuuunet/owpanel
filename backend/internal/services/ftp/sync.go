package ftp

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func syncPureFTP(username, password, home string, add bool) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := ensurePureFTPD(); err != nil {
		return err
	}
	if _, err := exec.LookPath("pure-pw"); err != nil {
		return fmt.Errorf("pure-ftpd 未安装，请在软件商店安装 Pure-FTPd")
	}
	home = filepath.Clean(home)
	if add {
		if err := os.MkdirAll(home, 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		args := []string{"useradd", username, "-u", "www-data", "-g", "www-data", "-d", home, "-m"}
		cmd := exec.Command("pure-pw", args...)
		cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("pure-pw useradd: %s", strings.TrimSpace(string(out)))
		}
	} else {
		out, err := exec.Command("pure-pw", "userdel", username, "-m").CombinedOutput()
		if err != nil && !strings.Contains(string(out), "Unknown user") {
			return fmt.Errorf("pure-pw userdel: %s", strings.TrimSpace(string(out)))
		}
	}
	return reloadPureFTPDB()
}

func syncPureFTPPassword(username, password string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	if err := ensurePureFTPD(); err != nil {
		return err
	}
	cmd := exec.Command("pure-pw", "passwd", username)
	cmd.Stdin = strings.NewReader(password + "\n" + password + "\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pure-pw passwd: %s", strings.TrimSpace(string(out)))
	}
	return reloadPureFTPDB()
}

func reloadPureFTPDB() error {
	if out, err := exec.Command("pure-pw", "mkdb").CombinedOutput(); err != nil {
		return fmt.Errorf("pure-pw mkdb: %s", strings.TrimSpace(string(out)))
	}
	_ = exec.Command("systemctl", "try-reload-or-restart", "pure-ftpd").Run()
	_ = exec.Command("systemctl", "try-reload-or-restart", "pure-ftpd@both").Run()
	return nil
}

func ensurePureFTPD() error {
	if _, err := exec.LookPath("pure-pw"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("apt-get"); err != nil {
		return nil
	}
	out, err := exec.Command("apt-get", "install", "-y", "pure-ftpd").CombinedOutput()
	if err != nil {
		return fmt.Errorf("安装 pure-ftpd: %s", strings.TrimSpace(string(out)))
	}
	_ = exec.Command("systemctl", "enable", "--now", "pure-ftpd").Run()
	return nil
}
