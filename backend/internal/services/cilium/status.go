package cilium

import (
	"strings"
)

type StatusResult struct {
	K3sInstalled      bool   `json:"k3s_installed"`
	K3sRunning        bool   `json:"k3s_running"`
	CiliumInstalled   bool   `json:"cilium_installed"`
	CiliumReady       bool   `json:"cilium_ready"`
	HostFirewallOn    bool   `json:"host_firewall_on"`
	HubbleEnabled     bool   `json:"hubble_enabled"`
	CiliumVersion     string `json:"cilium_version,omitempty"`
	ReadyPods         int    `json:"ready_pods"`
	TotalPods         int    `json:"total_pods"`
	HubbleUIPort      int    `json:"hubble_ui_port"`
	HubbleUIHint      string `json:"hubble_ui_hint,omitempty"`
	KernelOK          bool   `json:"kernel_ok"`
	Hint              string `json:"hint,omitempty"`
	LinuxOnly         bool   `json:"linux_only"`
}

func (s *Service) Status() (*StatusResult, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	out := &StatusResult{
		HostFirewallOn: cfg.HostFirewallEnabled,
		HubbleEnabled:  cfg.HubbleEnabled,
		HubbleUIPort:   12000,
		LinuxOnly:      !s.linuxHost(),
	}
	if !s.linuxHost() {
		out.Hint = "Cilium eBPF 防火墙需在 Linux 服务器上运行"
		return out, nil
	}

	out.K3sRunning = s.k3sRunning()
	out.K3sInstalled = out.K3sRunning || s.appInstalled("k3s")
	if !out.K3sInstalled {
		out.Hint = "请先在软件商店安装 k3s，或使用下方一键安装"
		return out, nil
	}
	if !out.K3sRunning {
		out.Hint = "k3s 已安装但未运行，请执行 systemctl start k3s"
		return out, nil
	}

	out.KernelOK = s.kernelOK()
	if !out.KernelOK {
		out.Hint = "建议内核 5.10+ 以使用 Cilium eBPF 完整功能"
	}

	ciliumInstalled, ready, readyN, totalN, version := s.ciliumPodStatus()
	out.CiliumInstalled = ciliumInstalled || s.appInstalled("cilium")
	out.CiliumReady = ready
	out.ReadyPods = readyN
	out.TotalPods = totalN
	out.CiliumVersion = version

	if !out.CiliumInstalled {
		out.Hint = "请安装 Cilium（软件商店 → 安全 → Cilium）或使用一键安装"
		return out, nil
	}
	if !out.CiliumReady {
		out.Hint = "Cilium 正在启动，请稍后刷新（kubectl get pods -n kube-system -l app.kubernetes.io/part-of=cilium）"
		return out, nil
	}

	if cfg.HubbleUIEnabled {
		out.HubbleUIHint = "kubectl port-forward -n kube-system svc/hubble-ui 12000:80"
	}
	if cfg.HostFirewallEnabled {
		if cfg.AuditMode {
			out.Hint = "Host Firewall 已启用（审计模式）。确认策略无误后可关闭审计模式"
		}
	} else {
		out.Hint = "可在下方开启 Cilium Host Firewall（eBPF 节点级防火墙）"
	}
	return out, nil
}

func (s *Service) ciliumPodStatus() (installed, ready bool, readyCount, total int, version string) {
	out, err := kubectl("get", "pods", "-n", "kube-system", "-l", "app.kubernetes.io/part-of=cilium", "-o", "wide", "--no-headers")
	if err != nil {
		return false, false, 0, 0, ""
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		total++
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[1] == "Running" && (strings.HasPrefix(fields[2], "1/1") || strings.HasPrefix(fields[2], "2/2") || strings.Contains(fields[2], "/")) {
				readyCount++
			}
	}
	installed = total > 0
	ready = installed && readyCount > 0 && readyCount >= total/2

	verOut, err := kubectl("get", "daemonset", "-n", "kube-system", "cilium", "-o", "jsonpath={.spec.template.spec.containers[0].image}")
	if err == nil {
		version = strings.TrimSpace(verOut)
		if i := strings.LastIndex(version, ":"); i >= 0 {
			version = version[i+1:]
		}
	}
	return
}
