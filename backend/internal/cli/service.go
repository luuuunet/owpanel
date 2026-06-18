package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/open-panel/open-panel/internal/database"
	"github.com/open-panel/open-panel/internal/secrets"
)

func RestartPanel() error  { return serviceAction("restart") }
func StopPanel() error     { return serviceAction("stop") }
func StartPanel() error    { return serviceAction("start") }
func ReloadPanel() error   { return serviceAction("reload") }

func serviceAction(action string) error {
	if runtime.GOOS == "linux" {
		if hasSystemctl() {
			name := serviceName()
			if action == "reload" {
				action = "restart"
			}
			cmd := exec.Command("systemctl", action, name)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("systemctl %s %s: %w", action, name, err)
			}
			printSuccess(fmt.Sprintf("Panel %s via systemd (%s)", action, name))
			return nil
		}
	}
	return fmt.Errorf("service control requires Linux with systemd (service: %s)", serviceName())
}

func hasSystemctl() bool {
	_, err := exec.LookPath("systemctl")
	return err == nil
}

func ClearPanelCache(ctx *Context) error {
	dirs := []string{
		filepath.Join(ctx.DataDir, "cache"),
		filepath.Join(ctx.DataDir, "tmp"),
	}
	var cleared int
	for _, d := range dirs {
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(d, e.Name())); err == nil {
				cleared++
			}
		}
	}
	printSuccess(fmt.Sprintf("Cleared %d cache item(s)", cleared))
	return nil
}

func RepairPanel(ctx *Context) error {
	fmt.Println("  Checking panel database and directories...")
	if err := os.MkdirAll(ctx.DataDir, 0755); err != nil {
		return err
	}
	for _, sub := range []string{"wwwroot", "backup", "logs", "server"} {
		if err := os.MkdirAll(filepath.Join(ctx.DataDir, sub), 0755); err != nil {
			return err
		}
	}
	dbCtx, err := NewContext()
	if err != nil {
		return fmt.Errorf("database check failed: %w", err)
	}
	if ans := strings.ToLower(strings.TrimSpace(readLine("  Reset admin password? (y/N): "))); ans == "y" || ans == "yes" {
		db, err := database.Init(dbCtx.DataDir)
		if err != nil {
			return err
		}
		pass, err := secrets.GeneratePassword(16)
		if err != nil {
			return fmt.Errorf("generate password: %w", err)
		}
		if err := database.ResetAdminPassword(db, "admin", pass); err != nil {
			return fmt.Errorf("reset admin password: %w", err)
		}
		credPath, _ := secrets.WriteInitialAdminCredentials(dbCtx.DataDir, "admin", pass)
		printSuccess("Admin password reset.")
		if credPath != "" {
			fmt.Printf("  Credentials saved to: %s\n", credPath)
		}
		fmt.Printf("  New password: %s\n", pass)
	}
	webDir := ctx.Cfg.WebDir
	if !filepath.IsAbs(webDir) {
		webDir = filepath.Join(ctx.Install, "web")
	}
	if webDir == "" {
		webDir = filepath.Join(ctx.Install, "web")
	}
	if st, err := os.Stat(filepath.Join(webDir, "index.html")); err != nil || st.IsDir() {
		fmt.Println(yellow("  ! Frontend web files missing — run: cd frontend && npm run build"))
	} else {
		printSuccess("Web files OK")
	}
	printSuccess("Repair complete. Restart the panel if issues persist (op restart).")
	return nil
}

func ClearSystemRubbish(ctx *Context) error {
	var freed int64
	logDir := filepath.Join(ctx.DataDir, "logs")
	_ = filepath.Walk(logDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".log") && info.Size() > 10*1024*1024 {
			freed += info.Size()
			_ = os.Truncate(path, 0)
		}
		return nil
	})
	printSuccess(fmt.Sprintf("Log cleanup done (~%d bytes truncated)", freed))
	return nil
}
