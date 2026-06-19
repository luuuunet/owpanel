package k8s

type SetupStep struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Done        bool   `json:"done"`
	Current     bool   `json:"current"`
	Action      string `json:"action,omitempty"`
}

type ChecklistItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Pass  bool   `json:"pass"`
	Level string `json:"level"`
	Hint  string `json:"hint,omitempty"`
}

type DashboardResult struct {
	Settings    ClusterSettings `json:"settings"`
	Status      *StatusResult   `json:"status"`
	HealthScore int             `json:"health_score"`
	SetupSteps  []SetupStep     `json:"setup_steps"`
	Checklist   []ChecklistItem `json:"checklist"`
}

func (s *Service) Dashboard() (*DashboardResult, error) {
	st, err := s.Status()
	if err != nil {
		return nil, err
	}

	sampleDeployed := s.sampleAppDeployed()
	var steps []SetupStep
	var checklist []ChecklistItem

	if st.ClusterMode == ModeStandard {
		kubeOk := s.kubeconfigExists()
		steps = []SetupStep{
			{
				Key: "kubeconfig", Title: "配置 kubeconfig", Description: "指向已有 K8s 集群的 admin 配置",
				Done: kubeOk && st.ClusterConnected, Current: !st.ClusterConnected, Action: "save_kubeconfig",
			},
			{
				Key: "verify", Title: "验证集群", Description: "节点与系统 Pod 就绪",
				Done: st.K8sReady, Current: st.ClusterConnected && !st.K8sReady, Action: "refresh",
			},
			{
				Key: "sample", Title: "示例应用（可选）", Description: "部署 nginx 验证工作负载",
				Done: sampleDeployed, Current: st.K8sReady && !sampleDeployed, Action: "deploy_sample",
			},
		}
		checklist = []ChecklistItem{
			{Key: "linux", Label: "Linux 服务器", Pass: !st.LinuxOnly, Level: "high", Hint: "kubectl 需 Linux"},
			{Key: "kubeconfig", Label: "kubeconfig 可访问", Pass: kubeOk, Level: "high"},
			{Key: "connected", Label: "集群已连接", Pass: st.ClusterConnected, Level: "high"},
			{Key: "nodes", Label: "节点 Ready", Pass: st.NodesTotal > 0 && st.NodesReady >= st.NodesTotal, Level: "high"},
			{Key: "system", Label: "系统 Pod 健康", Pass: st.SystemPodsTotal > 0 && st.SystemPodsReady >= st.SystemPodsTotal, Level: "high"},
			{Key: "sample", Label: "示例 nginx（可选）", Pass: sampleDeployed, Level: "low"},
		}
	} else {
		steps = []SetupStep{
			{
				Key: "k3s", Title: "安装 K3s", Description: "轻量 Kubernetes（兼容标准 K8s API）",
				Done: st.K3sRunning, Current: !st.K3sRunning, Action: "install_k3s",
			},
			{
				Key: "verify", Title: "验证集群", Description: "节点与系统 Pod 就绪",
				Done: st.K8sReady, Current: st.K3sRunning && !st.K8sReady, Action: "refresh",
			},
			{
				Key: "sample", Title: "示例应用（可选）", Description: "部署 nginx 验证工作负载",
				Done: sampleDeployed, Current: st.K8sReady && !sampleDeployed, Action: "deploy_sample",
			},
		}
		checklist = []ChecklistItem{
			{Key: "linux", Label: "Linux 服务器", Pass: !st.LinuxOnly, Level: "high", Hint: "K3s 需 Linux"},
			{Key: "k3s", Label: "K3s 运行中", Pass: st.K3sRunning, Level: "high"},
			{Key: "nodes", Label: "节点 Ready", Pass: st.NodesTotal > 0 && st.NodesReady >= st.NodesTotal, Level: "high"},
			{Key: "system", Label: "系统 Pod 健康", Pass: st.SystemPodsTotal > 0 && st.SystemPodsReady >= st.SystemPodsTotal, Level: "high"},
			{Key: "sample", Label: "示例 nginx（可选）", Pass: sampleDeployed, Level: "low"},
		}
	}

	score := 0
	if !st.LinuxOnly {
		score += 15
	}
	if st.ClusterConnected {
		score += 30
	}
	if st.NodesTotal > 0 && st.NodesReady >= st.NodesTotal {
		score += 25
	}
	if st.SystemPodsTotal > 0 && st.SystemPodsReady >= st.SystemPodsTotal {
		score += 20
	}
	if sampleDeployed {
		score += 10
	}

	return &DashboardResult{
		Settings:    s.GetSettings(),
		Status:      st,
		HealthScore: score,
		SetupSteps:  steps,
		Checklist:   checklist,
	}, nil
}
