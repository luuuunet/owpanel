package dataplatform

type PipelineItem struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Path   string `json:"path"`
}

type OrchestrationSummary struct {
	ClusterNodes    int            `json:"cluster_nodes"`
	ClusterOnline   int            `json:"cluster_online"`
	LBActive        int            `json:"lb_active"`
	K8sReady        bool           `json:"k8s_ready"`
	K8sNodes        int            `json:"k8s_nodes"`
	ComposeTotal    int            `json:"compose_total"`
	ComposeRunning  int            `json:"compose_running"`
	Pipelines       []PipelineItem `json:"pipelines"`
	DevOpsPath      string         `json:"devops_path"`
	Hint            string         `json:"hint,omitempty"`
}

func (s *Service) Orchestration() OrchestrationSummary {
	out := OrchestrationSummary{DevOpsPath: "/devops"}
	if s.cluster != nil {
		if co, err := s.cluster.Overview(); err == nil {
			out.ClusterNodes = co.NodeTotal
			out.ClusterOnline = co.NodeOnline
			out.LBActive = co.LBActive
		}
	}
	if s.k8s != nil {
		if st, err := s.k8s.Status(); err == nil && st != nil {
			out.K8sReady = st.K8sReady
			out.K8sNodes = st.NodesTotal
		}
	}
	if s.compose != nil {
		if list, err := s.compose.List(); err == nil {
			out.ComposeTotal = len(list)
			for _, c := range list {
				if c.LiveStatus == "running" {
					out.ComposeRunning++
				}
				out.Pipelines = append(out.Pipelines, PipelineItem{
					Name: c.Name, Type: "compose",
					Status: c.LiveStatus, Path: "/compose",
				})
			}
		}
	}
	if out.K8sReady {
		out.Pipelines = append([]PipelineItem{{
			Name: "Kubernetes", Type: "k8s", Status: "running", Path: "/k8s",
		}}, out.Pipelines...)
	}
	if out.ClusterOnline > 1 {
		out.Pipelines = append(out.Pipelines, PipelineItem{
			Name: "Multi-node cluster", Type: "cluster", Status: "active", Path: "/cluster",
		})
	}
	out.Pipelines = append(out.Pipelines, PipelineItem{
		Name: "DevOps CI/CD", Type: "devops", Status: "available", Path: "/devops",
	})
	switch {
	case out.ClusterOnline == 0 && !out.K8sReady:
		out.Hint = "Add cluster nodes or install K3s to orchestrate AI workloads across servers."
	case out.ComposeRunning == 0 && out.K8sReady:
		out.Hint = "K8s ready — deploy AI inference as Deployments or use Compose for single-node stacks."
	case out.ComposeRunning > 0 && out.K8sReady:
		out.Hint = "Hybrid orchestration: Compose for data plane (Milvus/VM), K8s for inference scaling."
	default:
		out.Hint = "Use DevOps center for CI/CD pipelines tying model deploy, vector sync, and monitoring."
	}
	return out
}
