package appstore

import (
	"github.com/open-panel/open-panel/internal/services/phpmyadmin"
)

func tryPhpMyAdminInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != "phpmyadmin" {
		return false, nil
	}
	svc := phpmyadmin.New(dataDir, nil)
	if err := svc.Install(installPath, version, 888); err != nil {
		return true, err
	}
	return true, nil
}

func tryPhpMyAdminUninstall(key, dataDir string) (bool, error) {
	if key != "phpmyadmin" {
		return false, nil
	}
	return true, phpmyadmin.New(dataDir, nil).Uninstall("")
}

func detectPhpMyAdminStatus(dataDir string, port int) string {
	if port <= 0 {
		port = 888
	}
	return phpmyadmin.New(dataDir, nil).Status(port)
}
