package settings

import (
	"path/filepath"
	"strings"
)

func defaultBackupPath(dataDir string) string {
	return filepath.Join(dataDir, "backup")
}

func defaultWebsitePath(dataDir string) string {
	return filepath.Join(dataDir, "wwwroot")
}

func defaultPaths(dataDir string) map[string]string {
	return map[string]string{
		"backup_path":  defaultBackupPath(dataDir),
		"website_path": defaultWebsitePath(dataDir),
	}
}

// DefaultBackupPath is the OS-aware fallback when settings are empty.
func DefaultBackupPath(dataDir string) string {
	return defaultBackupPath(dataDir)
}

// DefaultWebsitePath is the fallback website root under the panel data directory.
func DefaultWebsitePath(dataDir string) string {
	return defaultWebsitePath(dataDir)
}

// DefaultSecurityLogPath is the WAF security log location under the panel data directory.
func DefaultSecurityLogPath(dataDir string) string {
	return filepath.Join(dataDir, "logs", "security.log")
}

// ServerInstallPath returns the panel-managed install directory for a server app key.
func ServerInstallPath(dataDir, key string) string {
	return filepath.Join(dataDir, "server", key)
}

// ServerPHPPath returns the panel-managed install directory for a PHP version key (e.g. php83).
func ServerPHPPath(dataDir, key string) string {
	return filepath.Join(dataDir, "server", key)
}

// AIInstallPath returns the panel-managed install directory for an AI app key.
func AIInstallPath(dataDir, key string) string {
	return filepath.Join(dataDir, "apps", key)
}

// DockerAppPath returns the panel-managed directory for a Docker catalog app.
func DockerAppPath(dataDir, key string) string {
	return filepath.Join(dataDir, "apps", key)
}

// ResolvePanelPath resolves catalog-relative paths (e.g. "server/nginx") against dataDir.
// Absolute system paths (e.g. /etc/nginx/nginx.conf) are returned unchanged.
// Stored absolute paths under /www/ are remapped to the panel data directory layout.
func ResolvePanelPath(dataDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
		switch {
		case p == "/www/wwwroot":
			return defaultWebsitePath(dataDir)
		case p == "/www/backup":
			return defaultBackupPath(dataDir)
		case strings.HasPrefix(p, "/www/wwwlogs/"):
			return filepath.Join(dataDir, "logs", strings.TrimPrefix(p, "/www/wwwlogs/"))
		case strings.HasPrefix(p, "/www/server/"):
			return filepath.Join(dataDir, "server", strings.TrimPrefix(p, "/www/server/"))
		case strings.HasPrefix(p, "/www/ai/"):
			return filepath.Join(dataDir, "ai", strings.TrimPrefix(p, "/www/ai/"))
		case strings.HasPrefix(p, "/www/apps/"):
			return filepath.Join(dataDir, "apps", strings.TrimPrefix(p, "/www/apps/"))
		default:
			return filepath.FromSlash(p)
		}
	}
	return filepath.Join(dataDir, filepath.FromSlash(p))
}
