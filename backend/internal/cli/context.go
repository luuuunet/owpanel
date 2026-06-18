package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/open-panel/open-panel/internal/auth"
	"github.com/open-panel/open-panel/internal/config"
	"github.com/open-panel/open-panel/internal/database"
	"github.com/open-panel/open-panel/internal/services/settings"
)

const systemdUnit = "open-panel"

type Context struct {
	Cfg      *config.Config
	DataDir  string
	Install  string
	Auth     *auth.Service
	Settings *settings.Service
}

func NewContext() (*Context, error) {
	ensurePanelEnv()
	cfg := config.Load()
	dataDir := cfg.DataDir
	if !filepath.IsAbs(dataDir) {
		if wd, err := os.Getwd(); err == nil {
			dataDir = filepath.Join(wd, dataDir)
		}
	}

	install := os.Getenv("OPEN_PANEL_HOME")
	if install == "" {
		install = filepath.Dir(dataDir)
	}

	db, err := database.Init(dataDir)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}

	return &Context{
		Cfg:      cfg,
		DataDir:  dataDir,
		Install:  install,
		Auth:     auth.NewService(db, cfg.JWTSecret),
		Settings: settings.NewServiceWithDataDir(db, dataDir),
	}, nil
}

func serviceName() string {
	if v := os.Getenv("OPEN_PANEL_SERVICE"); v != "" {
		return v
	}
	return systemdUnit
}

var envLineRe = regexp.MustCompile(`^Environment=(OPEN_PANEL_[A-Z_]+)=(.*)$`)

func ensurePanelEnv() {
	if os.Getenv("OPEN_PANEL_DATA") != "" {
		return
	}
	unit := filepath.Join("/etc/systemd/system", serviceName()+".service")
	data, err := os.ReadFile(unit)
	if err != nil {
		if _, err := os.Stat("/opt/open-panel/data"); err == nil {
			os.Setenv("OPEN_PANEL_DATA", "/opt/open-panel/data")
			os.Setenv("OPEN_PANEL_HOME", "/opt/open-panel")
		}
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		m := envLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) == 3 {
			os.Setenv(m[1], strings.Trim(m[2], `"`))
		}
	}
}

func readLine(prompt string) string {
	fmt.Print(prompt)
	var s string
	fmt.Scanln(&s)
	return s
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	var s string
	fmt.Scanln(&s)
	return s
}

func readInt(prompt string, def int) int {
	s := readLine(prompt)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func pause() {
	fmt.Print("\nPress Enter to continue...")
	fmt.Scanln()
}
