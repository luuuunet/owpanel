package website

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type SiteDiagnosticBundle struct {
	SiteID          uint     `json:"site_id"`
	Domain          string   `json:"domain"`
	RootPath        string   `json:"root_path"`
	Status          string   `json:"status"`
	PhpVersion      string   `json:"php_version"`
	PHP             bool     `json:"php"`
	SSL             bool     `json:"ssl"`
	ForceHTTPS      bool     `json:"force_https"`
	IndexFiles      string   `json:"index_files"`
	NginxConf       string   `json:"nginx_conf"`
	RootExists      bool     `json:"root_exists"`
	RootWritable    bool     `json:"root_writable"`
	HasIndexPHP     bool     `json:"has_index_php"`
	HasIndexHTML    bool     `json:"has_index_html"`
	WebServerActive string   `json:"web_server_active"`
	WebServerRunning bool    `json:"web_server_running"`
	AliasDomains    []string `json:"alias_domains"`
	AccessLogTail   string   `json:"access_log_tail"`
	ErrorLogTail    string   `json:"error_log_tail"`
	NginxConfSnippet string  `json:"nginx_conf_snippet"`
	Issues          []string `json:"issues"`
}

func (s *Service) CollectDiagnostics(siteID uint) (*SiteDiagnosticBundle, error) {
	site, err := s.Get(siteID)
	if err != nil {
		return nil, err
	}
	b := &SiteDiagnosticBundle{
		SiteID:     siteID,
		Domain:     site.Domain,
		RootPath:   site.RootPath,
		Status:     site.Status,
		PhpVersion: site.PhpVersion,
		PHP:        site.PHP,
		SSL:        site.SSL,
		ForceHTTPS: site.ForceHTTPS,
		IndexFiles: site.IndexFiles,
		NginxConf:  site.NginxConf,
	}
	for _, a := range site.Aliases {
		if a.Domain != "" {
			b.AliasDomains = append(b.AliasDomains, a.Domain)
		}
	}
	if site.RootPath != "" {
		if st, err := os.Stat(site.RootPath); err == nil {
			b.RootExists = st.IsDir()
			b.RootWritable = st.Mode().Perm()&0200 != 0
		} else {
			b.Issues = append(b.Issues, "网站根目录不存在: "+site.RootPath)
		}
		if b.RootExists {
			b.HasIndexPHP = fileExists(filepath.Join(site.RootPath, "index.php"))
			b.HasIndexHTML = fileExists(filepath.Join(site.RootPath, "index.html"))
			if site.PHP && !b.HasIndexPHP && !b.HasIndexHTML {
				b.Issues = append(b.Issues, "PHP 站点根目录缺少 index.php / index.html")
			}
		}
	}
	if site.Status == "stopped" {
		b.Issues = append(b.Issues, "站点当前为停止状态")
	}
	if logs, err := s.SiteLogs(siteID, 80); err == nil {
		if v, ok := logs["access_tail"].(string); ok {
			b.AccessLogTail = trimLog(v, 12000)
		}
		if v, ok := logs["error_tail"].(string); ok {
			b.ErrorLogTail = trimLog(v, 12000)
			if strings.TrimSpace(v) != "" {
				b.Issues = append(b.Issues, "错误日志中有近期记录")
			}
		}
	}
	if site.NginxConf != "" {
		if data, err := os.ReadFile(site.NginxConf); err == nil {
			b.NginxConfSnippet = trimLog(string(data), 8000)
		}
	} else {
		b.Issues = append(b.Issues, "未找到 Nginx 配置文件路径")
	}
	if s.ws != nil {
		if ov, err := s.ws.Overview(); err == nil {
			b.WebServerActive = ov.Active
			for _, srv := range ov.Servers {
				if srv.Key == ov.Active {
					b.WebServerRunning = srv.Status == "running"
					break
				}
			}
			if !b.WebServerRunning {
				b.Issues = append(b.Issues, "Web 服务器未运行")
			}
		}
	}
	return b, nil
}

func (s *Service) DiagnosticJSON(siteID uint) (string, error) {
	b, err := s.CollectDiagnostics(siteID)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Service) ApplyRepairAction(siteID uint, action string) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	switch action {
	case "create_root_dir":
		if site.RootPath == "" {
			return fmt.Errorf("未配置网站根目录")
		}
		return os.MkdirAll(site.RootPath, 0755)
	case "fix_dir_permissions":
		return fixDirPermissions(site.RootPath)
	case "ensure_index_files":
		idx := strings.TrimSpace(site.IndexFiles)
		if idx == "" {
			if site.PHP || (site.PhpVersion != "" && site.PhpVersion != "static") {
				idx = "index.php index.html index.htm"
			} else {
				idx = "index.html index.htm index.php"
			}
			return s.db.Model(site).Update("index_files", idx).Error
		}
		return nil
	case "start_site":
		if site.Status == "running" {
			return nil
		}
		_, err := s.ToggleSite(siteID, "running")
		return err
	case "start_php_fpm":
		return startPHPFPM(site.PhpVersion)
	case "apply_vhost":
		return s.ApplyVhost(siteID)
	case "reload_webserver":
		if s.ws == nil {
			return fmt.Errorf("web 服务器管理器不可用")
		}
		active := site.WebServer
		if active == "" {
			if ov, err := s.ws.Overview(); err == nil && ov.Active != "" {
				active = ov.Active
			} else {
				active = "nginx"
			}
		}
		return s.ws.Reload(active)
	default:
		return fmt.Errorf("不支持的修复动作: %s", action)
	}
}

func fixDirPermissions(root string) error {
	if root == "" {
		return fmt.Errorf("根目录为空")
	}
	if _, err := os.Stat(root); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		return nil
	}
	_ = exec.Command("find", root, "-type", "d", "-exec", "chmod", "755", "{}", "+").Run()
	_ = exec.Command("find", root, "-type", "f", "-exec", "chmod", "644", "{}", "+").Run()
	return nil
}

func startPHPFPM(version string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	ver := strings.TrimSpace(version)
	if ver == "" || ver == "static" {
		ver = "8.3"
	}
	svc := "php" + ver + "-fpm"
	if out, err := exec.Command("systemctl", "start", svc).CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", svc, strings.TrimSpace(string(out)))
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func trimLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return "...(truncated)\n" + s[len(s)-max:]
}
