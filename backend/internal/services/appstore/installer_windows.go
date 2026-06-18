package appstore

import (
	"os/exec"
	"runtime"
	"strings"
)

var windowsServiceNames = map[string]string{
	"nginx": "nginx", "mysql": "MySQL80", "mariadb": "MariaDB",
	"postgresql": "postgresql-x64-16", "redis": "Redis",
	"mongodb": "MongoDB", "apache": "Apache2.4", "memcached": "memcached",
	"docker": "com.docker.service", "php83": "php-cgi-8.3",
}

var windowsProcessNames = map[string][]string{
	"nginx": {"nginx.exe"}, "openresty": {"nginx.exe", "openresty.exe"},
	"apache": {"httpd.exe"}, "mysql": {"mysqld.exe"}, "mariadb": {"mysqld.exe"},
	"postgresql": {"postgres.exe"}, "redis": {"redis-server.exe"},
	"mongodb": {"mongod.exe"}, "memcached": {"memcached.exe"},
	"php83": {"php-cgi.exe"}, "php82": {"php-cgi.exe"}, "php81": {"php-cgi.exe"}, "php74": {"php-cgi.exe"},
	"nodejs20": {"node.exe"}, "nodejs18": {"node.exe"},
}

func detectWindowsServiceStatus(key, linuxSvc string) string {
	svc := linuxSvc
	if alt, ok := windowsServiceNames[key]; ok {
		svc = alt
	}
	if svc != "" {
		out, err := exec.Command("sc", "query", svc).CombinedOutput()
		if err == nil {
			text := string(out)
			if strings.Contains(text, "RUNNING") {
				return "running"
			}
			if strings.Contains(text, "STOPPED") {
				return "stopped"
			}
		}
	}
	if names, ok := windowsProcessNames[key]; ok {
		for _, proc := range names {
			if processExists(proc) {
				return "running"
			}
		}
	}
	return "stopped"
}

func processExists(name string) bool {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name, "/NH").CombinedOutput()
		if err != nil {
			return false
		}
		return strings.Contains(strings.ToLower(string(out)), strings.ToLower(name))
	}
	out, err := exec.Command("pgrep", "-x", name).CombinedOutput()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}
