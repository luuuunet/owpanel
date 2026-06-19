package k8s

import (
	"os"
	"strings"
)

const (
	SettingClusterMode      = "k8s_cluster_mode"
	SettingKubeconfigPath   = "k8s_kubeconfig_path"
	ModeK3s                 = "k3s"
	ModeStandard            = "standard"
	DefaultStandardKubeconfig = "/root/.kube/config"
)

type ClusterSettings struct {
	ClusterMode    string `json:"cluster_mode"`
	KubeconfigPath string `json:"kubeconfig_path"`
}

func (s *Service) ClusterMode() string {
	if s.settings == nil {
		return ModeK3s
	}
	all, err := s.settings.GetAll()
	if err != nil {
		return ModeK3s
	}
	if strings.TrimSpace(all[SettingClusterMode]) == ModeStandard {
		return ModeStandard
	}
	return ModeK3s
}

func (s *Service) KubeconfigPath() string {
	path := DefaultStandardKubeconfig
	if s.settings != nil {
		all, err := s.settings.GetAll()
		if err == nil {
			if p := strings.TrimSpace(all[SettingKubeconfigPath]); p != "" {
				path = p
			}
		}
	}
	return path
}

func (s *Service) GetSettings() ClusterSettings {
	return ClusterSettings{
		ClusterMode:    s.ClusterMode(),
		KubeconfigPath: s.KubeconfigPath(),
	}
}

func (s *Service) UpdateSettings(req ClusterSettings) error {
	mode := strings.TrimSpace(req.ClusterMode)
	if mode != ModeStandard {
		mode = ModeK3s
	}
	kubeconfig := strings.TrimSpace(req.KubeconfigPath)
	if kubeconfig == "" {
		kubeconfig = DefaultStandardKubeconfig
	}
	if s.settings == nil {
		return nil
	}
	return s.settings.Update(map[string]string{
		SettingClusterMode:    mode,
		SettingKubeconfigPath: kubeconfig,
	})
}

func (s *Service) kubeconfigExists() bool {
	_, err := os.Stat(s.KubeconfigPath())
	return err == nil
}

func (s *Service) clusterConnected() bool {
	if !s.linuxHost() {
		return false
	}
	if s.ClusterMode() == ModeK3s {
		return s.k3sRunning()
	}
	if !s.kubeconfigExists() {
		return false
	}
	_, err := s.kubectl("cluster-info")
	return err == nil
}
