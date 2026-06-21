package appstore

import "runtime"

func tryMongoDBInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != "mongodb" {
		return false, nil
	}
	if runtime.GOOS != "linux" {
		return false, nil
	}
	_ = version
	_ = installPath
	_ = dataDir
	logInstallLine("MongoDB：使用 stack 安装脚本（官方源 / Docker 回退）…")
	return true, runStackFallback("mongodb")
}
