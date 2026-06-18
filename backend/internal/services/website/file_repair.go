package website

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var blockedWritePatterns = []string{
	"wp-config.php",
	".env",
	"config.php",
	"database.php",
	"settings.php",
}

// WriteSiteFile creates or updates a file under the site root (AI repair use).
func (s *Service) WriteSiteFile(siteID uint, relPath, content string) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	if site.RootPath == "" {
		return fmt.Errorf("未配置网站根目录")
	}
	clean := filepath.Clean(strings.TrimPrefix(strings.ReplaceAll(relPath, "\\", "/"), "/"))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "..") {
		return fmt.Errorf("非法相对路径")
	}
	lower := strings.ToLower(clean)
	for _, blocked := range blockedWritePatterns {
		if strings.Contains(lower, blocked) {
			return fmt.Errorf("禁止修改敏感文件: %s", clean)
		}
	}
	rootAbs, err := filepath.Abs(site.RootPath)
	if err != nil {
		return err
	}
	full := filepath.Join(rootAbs, clean)
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return err
	}
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs, rootAbs+string(os.PathSeparator)) {
		return fmt.Errorf("路径超出网站根目录")
	}
	if err := os.MkdirAll(filepath.Dir(fullAbs), 0755); err != nil {
		return err
	}
	return os.WriteFile(fullAbs, []byte(content), 0644)
}
