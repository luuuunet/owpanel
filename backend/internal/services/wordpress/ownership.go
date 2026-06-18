package wordpress

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func (s *Service) ensureWPSiteOwnership(root string, logger *DeployLogger) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	root = filepath.Clean(root)
	if root == "" || root == "/" {
		return nil
	}
	if _, err := os.Stat(root); err != nil {
		return err
	}
	if _, err := exec.LookPath("chown"); err != nil {
		return nil
	}
	out, err := exec.Command("chown", "-R", "www-data:www-data", root).CombinedOutput()
	if err != nil {
		if logger != nil {
			logger.Warn("设置站点目录权限失败: " + string(out))
		}
		return err
	}
	if logger != nil {
		logger.Info("✓ 站点目录已设为 www-data 可写")
	}
	return nil
}
