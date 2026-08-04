package appstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luuuunet/owpanel/internal/services/php"
)

// phpPanelInstalled reports whether the panel installed/managed this PHP version (marker file).
func phpPanelInstalled(key, dataDir string) bool {
	if key == "phpmyadmin" || !strings.HasPrefix(key, "php") {
		return false
	}
	if fileExists(filepath.Join(dataDir, "php", key, ".owpanel-installed")) {
		return true
	}
	if fileExists(filepath.Join(dataDir, "server", key, ".owpanel-installed")) {
		return true
	}
	return false
}

// phpSystemPresent reports whether the host has this PHP version (FPM /etc/php or binary).
func phpSystemPresent(key, dataDir string) bool {
	if !IsPHPKey(key) {
		return false
	}
	ver := php.VersionFromKey(key)
	if ver != "" {
		for _, v := range php.DiscoverInstalledVersions() {
			if v == ver {
				return true
			}
		}
	}
	st := php.NewManager(dataDir).Status(key)
	return strings.TrimSpace(st.Binary) != ""
}

// phpPresent is true when the panel can manage this PHP (marker, system package, or live binary).
func phpPresent(key, dataDir string) bool {
	return phpPanelInstalled(key, dataDir) || phpSystemPresent(key, dataDir)
}

func phpCatalogKeyForVersion(ver string) string {
	ver = strings.TrimSpace(ver)
	if ver == "" {
		return ""
	}
	for _, item := range mergedCatalog() {
		if IsPHPKey(item.Key) && item.Version == ver {
			return item.Key
		}
	}
	return "php" + strings.ReplaceAll(ver, ".", "")
}

func writePHPAdoptMarker(key, version, dataDir string) {
	runtimeDir := filepath.Join(dataDir, "php", key)
	_ = os.MkdirAll(runtimeDir, 0755)
	marker := filepath.Join(runtimeDir, ".owpanel-installed")
	if fileExists(marker) {
		return
	}
	port := php.PortForVersion(version)
	_ = os.WriteFile(marker, []byte(fmt.Sprintf("version=%s\nport=%d\nadopted=1\n", version, port)), 0644)
}

func (s *Service) ensurePHPAppInstalled(key string) error {
	app, err := s.Get(key)
	if err != nil {
		return err
	}
	ver := php.VersionFromKey(key)
	if ver == "" {
		ver = app.Version
	}
	if !phpPanelInstalled(key, s.dataDir) && phpSystemPresent(key, s.dataDir) {
		writePHPAdoptMarker(key, ver, s.dataDir)
	}
	if app.Installed {
		return nil
	}
	status := "stopped"
	if php.NewManager(s.dataDir).Status(key).Running {
		status = "running"
	}
	return s.db.Model(&app).Updates(map[string]interface{}{
		"installed":     true,
		"status":        status,
		"version":       ver,
		"install_error": "",
	}).Error
}

func (s *Service) reconcilePHPInstallRecords() {
	// Adopt system-discovered PHP into the software catalog (fixes Installed tab missing PHP).
	for _, ver := range php.DiscoverInstalledVersions() {
		key := phpCatalogKeyForVersion(ver)
		if key == "" || !IsPHPKey(key) {
			continue
		}
		app, err := s.Get(key)
		if err != nil {
			continue
		}
		writePHPAdoptMarker(key, ver, s.dataDir)
		status := "stopped"
		if php.NewManager(s.dataDir).Status(key).Running {
			status = "running"
		}
		updates := map[string]interface{}{
			"installed":     true,
			"status":        status,
			"install_error": "",
		}
		if app.Version == "" {
			updates["version"] = ver
		}
		if !app.Installed || app.Status != status {
			_ = s.db.Model(&app).Updates(updates).Error
			s.InvalidateLiveStatus(key)
		}
	}

	// Only clear DB install flag when PHP is truly gone from the host.
	for _, item := range mergedCatalog() {
		key := item.Key
		if !IsPHPKey(key) {
			continue
		}
		app, err := s.Get(key)
		if err != nil || !app.Installed || IsSimulatedInstall(key, s.dataDir) {
			continue
		}
		if phpPresent(key, s.dataDir) {
			continue
		}
		_ = s.db.Model(&app).Updates(map[string]interface{}{
			"installed":     false,
			"status":        "stopped",
			"install_error": "",
		}).Error
		s.InvalidateLiveStatus(key)
	}
}

// phpVersionInstalledForListing is true when the panel should show this PHP in runtime/management UI.
func (s *Service) phpVersionInstalledForListing(key string) bool {
	if phpPresent(key, s.dataDir) {
		return true
	}
	app, err := s.Get(key)
	return err == nil && app.Installed && !IsSimulatedInstall(key, s.dataDir)
}

func phpBinaryForKey(key, dataDir string) string {
	st := php.NewManager(dataDir).Status(key)
	return st.Binary
}
