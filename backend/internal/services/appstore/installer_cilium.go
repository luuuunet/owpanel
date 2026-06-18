package appstore

import (
	"fmt"
	"runtime"
)

func tryCiliumInstall(key, version, installPath, dataDir string) (bool, error) {
	if key != "cilium" {
		return false, nil
	}
	_ = version
	_ = installPath
	_ = dataDir
	if runtime.GOOS != "linux" {
		return true, fmt.Errorf("Cilium 仅支持 Linux 服务器")
	}
	if !K3sRunning() {
		return true, fmt.Errorf("请先安装 k3s（软件商店 → DevOps → K3s）")
	}
	return true, InstallCiliumHelm(true, true, true)
}

func InstallCiliumHelm(hostFirewall, hubble, hubbleUI bool) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("Cilium 仅支持 Linux")
	}
	if !K3sRunning() {
		return fmt.Errorf("k3s 未运行")
	}
	logInstallLine("检查 Helm …")
	_ = runCommand("bash", "-c", "command -v helm >/dev/null || curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash")

	env := "export KUBECONFIG=" + K3sKubeConfig
	steps := []string{
		env + " && helm repo add cilium https://helm.cilium.io/ 2>/dev/null || true",
		env + " && helm repo update",
	}
	for _, step := range steps {
		if err := runCommand("bash", "-c", step); err != nil {
			return fmt.Errorf("helm repo: %w", err)
		}
	}

	setFlags := "--set operator.replicas=1 --set kubeProxyReplacement=false"
	if hostFirewall {
		setFlags += " --set hostFirewall.enabled=true"
	}
	if hubble {
		setFlags += " --set hubble.enabled=true --set hubble.relay.enabled=true"
	}
	if hubbleUI {
		setFlags += " --set hubble.ui.enabled=true"
	}

	install := env + " && helm upgrade --install cilium cilium/cilium --namespace kube-system " + setFlags + " --wait --timeout 15m"
	logInstallLine("正在通过 Helm 安装 Cilium …")
	if err := runCommand("bash", "-c", install); err != nil {
		return fmt.Errorf("Cilium Helm 安装失败: %w", err)
	}
	logInstallLine("Cilium 安装完成")
	return nil
}

func RunCiliumInstall(dataDir string) error {
	ok, err := tryCiliumInstall("cilium", "latest", "", dataDir)
	if !ok {
		return fmt.Errorf("cilium installer unavailable")
	}
	return err
}
