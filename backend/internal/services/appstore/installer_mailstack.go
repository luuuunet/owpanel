package appstore

import "fmt"

func tryMailStackInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != "mail-server" {
		return false, nil
	}
	_ = version
	_ = installPath
	_ = dataDir
	if installService == nil || installService.mailStack == nil {
		return true, fmt.Errorf("邮件服务未初始化")
	}
	return true, installService.mailStack.InstallStack()
}

func tryMailStackUninstall(key, dataDir string) (bool, error) {
	if key != "mail-server" {
		return false, nil
	}
	_ = dataDir
	if installService == nil || installService.mailStack == nil {
		return true, fmt.Errorf("邮件服务未初始化")
	}
	if err := installService.mailStack.UninstallStack(); err != nil {
		return true, err
	}
	installService.syncMailStackRecords(false)
	return true, nil
}
