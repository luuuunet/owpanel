package k8s

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/luuuunet/owpanel/internal/services/appstore"
)

func (s *Service) clusterListReady() bool {
	if !s.linuxHost() {
		return false
	}
	if s.ClusterMode() == ModeStandard {
		return s.kubeconfigExists() && s.clusterConnected()
	}
	return appstore.K3sRunning()
}

func (s *Service) kubectl(args ...string) (string, error) {
	if !s.linuxHost() {
		return "", fmt.Errorf("K8s 需要 Linux 服务器")
	}

	kubeconfig := appstore.K3sKubeConfig
	if s.ClusterMode() == ModeStandard {
		kubeconfig = s.KubeconfigPath()
		if _, err := os.Stat(kubeconfig); err != nil {
			return "", fmt.Errorf("kubeconfig 不存在: %s", kubeconfig)
		}
	} else if !appstore.K3sRunning() {
		return "", fmt.Errorf("k3s 未运行")
	}

	cmd := exec.Command("kubectl", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
