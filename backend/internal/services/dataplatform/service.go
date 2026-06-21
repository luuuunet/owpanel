package dataplatform

import (
	"os/exec"
	"strings"

	"github.com/luuuunet/owpanel/internal/services/aihub"
	"github.com/luuuunet/owpanel/internal/services/appstore"
	"github.com/luuuunet/owpanel/internal/services/autops"
	"github.com/luuuunet/owpanel/internal/services/cilium"
	"github.com/luuuunet/owpanel/internal/services/cluster"
	"github.com/luuuunet/owpanel/internal/services/compose"
	"github.com/luuuunet/owpanel/internal/services/k8s"
	"github.com/luuuunet/owpanel/internal/services/logs"
	"github.com/luuuunet/owpanel/internal/services/settings"
)

type Service struct {
	dataDir  string
	appstore *appstore.Service
	cilium   *cilium.Service
	settings *settings.Service
	k8s      *k8s.Service
	aihub    *aihub.Service
	cluster  *cluster.Service
	logs     *logs.Service
	autops   *autops.Service
	compose  *compose.Service
}

func NewService(
	dataDir string,
	appSvc *appstore.Service,
	ciliumSvc *cilium.Service,
	settingsSvc *settings.Service,
	k8sSvc *k8s.Service,
	aihubSvc *aihub.Service,
	clusterSvc *cluster.Service,
	logsSvc *logs.Service,
	autopsSvc *autops.Service,
	composeSvc *compose.Service,
) *Service {
	return &Service{
		dataDir:  dataDir,
		appstore: appSvc,
		cilium:   ciliumSvc,
		settings: settingsSvc,
		k8s:      k8sSvc,
		aihub:    aihubSvc,
		cluster:  clusterSvc,
		logs:     logsSvc,
		autops:   autopsSvc,
		compose:  composeSvc,
	}
}

type QuickLink struct {
	Key   string `json:"key"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Icon  string `json:"icon"`
}

type CloudNativeSummary struct {
	K8sReady        bool   `json:"k8s_ready"`
	K3sRunning      bool   `json:"k3s_running"`
	NodesReady      int    `json:"nodes_ready"`
	NodesTotal      int    `json:"nodes_total"`
	CiliumReady     bool   `json:"cilium_ready"`
	DockerRunning   bool   `json:"docker_running"`
	VectorDBRunning int    `json:"vector_db_running"`
	MetricsRunning  int    `json:"metrics_running"`
	StorageRunning  int    `json:"storage_running"`
	Hint            string `json:"hint,omitempty"`
}

type AIInfraSummary struct {
	HFInstalled      bool   `json:"hf_installed"`
	HFRunning        bool   `json:"hf_running"`
	VectorDBRunning  int    `json:"vector_db_running"`
	VectorDBTotal    int    `json:"vector_db_total"`
	ModelWeightBytes int64  `json:"model_weight_bytes"`
	ModelWeightHuman string `json:"model_weight_human"`
	RAGReady         bool   `json:"rag_ready"`
	Hint             string `json:"hint,omitempty"`
}

type Overview struct {
	Title          string               `json:"title"`
	Subtitle       string               `json:"subtitle"`
	HealthScore    int                  `json:"health_score"`
	CloudNative    CloudNativeSummary   `json:"cloud_native"`
	AIInfra        AIInfraSummary       `json:"ai_infra"`
	LLMOps         LLMOpsSummary        `json:"llmops"`
	DataOps        DataOpsSummary       `json:"dataops"`
	AIOps          AIOpsSummary         `json:"aiops"`
	SecOps         SecOpsSummary        `json:"secops"`
	Orchestration  OrchestrationSummary `json:"orchestration"`
	QuickLinks     []QuickLink          `json:"quick_links"`
	VectorDBs      []VectorEngineStatus `json:"vector_dbs"`
	MetricsStores  []MetricsEngineStatus `json:"metrics_stores"`
	ModelWeights   WeightsSummary       `json:"model_weights"`
	SecurityIntel  SecurityIntelSummary `json:"security_intel"`
	StorageMeta    []StorageEngineMeta  `json:"storage_meta"`
}

func (s *Service) Overview() Overview {
	llm := s.LLMOps()
	data := s.DataOps()
	aiops := s.AIOps()
	sec := s.SecOps()
	orch := s.Orchestration()
	vectors := data.VectorDBs
	metrics := aiops.MetricsStores
	weights := s.WeightsSummary()
	storage := data.StorageMeta
	cn := s.CloudNativeSummary(vectors, metrics, storage)
	ai := s.AIInfraSummary(vectors, weights)
	score := computeInfraHealthScore(cn, ai, sec.SecurityIntelSummary)
	if aiops.HealthScore > 0 {
		score = (score + aiops.HealthScore) / 2
	}
	return Overview{
		Title:         "Cloud Native & AI Infrastructure Hub",
		Subtitle:      "LLMOps · DataOps · AIOps · SecOps · Orchestration",
		HealthScore:   score,
		CloudNative:   cn,
		AIInfra:       ai,
		LLMOps:        llm,
		DataOps:       data,
		AIOps:         aiops,
		SecOps:        sec,
		Orchestration: orch,
		QuickLinks:    defaultQuickLinks(),
		VectorDBs:     vectors,
		MetricsStores: metrics,
		ModelWeights:  weights,
		SecurityIntel: sec.SecurityIntelSummary,
		StorageMeta:   storage,
	}
}

func (s *Service) CloudNativeSummary(vectors []VectorEngineStatus, metrics []MetricsEngineStatus, storage []StorageEngineMeta) CloudNativeSummary {
	out := CloudNativeSummary{}
	if s.k8s != nil {
		if st, err := s.k8s.Status(); err == nil && st != nil {
			out.K8sReady = st.K8sReady
			out.K3sRunning = st.K3sRunning
			out.NodesReady = st.NodesReady
			out.NodesTotal = st.NodesTotal
			out.Hint = st.Hint
		}
	}
	if s.cilium != nil {
		if st, err := s.cilium.Status(); err == nil && st != nil {
			out.CiliumReady = st.CiliumReady
		}
	}
	out.DockerRunning = dockerEngineUp()
	for _, v := range vectors {
		if v.Running {
			out.VectorDBRunning++
		}
	}
	for _, m := range metrics {
		if m.Running {
			out.MetricsRunning++
		}
	}
	for _, st := range storage {
		if st.Running {
			out.StorageRunning++
		}
	}
	if !out.K8sReady && !out.K3sRunning && out.Hint == "" {
		out.Hint = "Deploy K3s/K8s from K8s cluster or install Cilium for cloud-native networking."
	} else if out.K8sReady && !out.CiliumReady && out.Hint == "" {
		out.Hint = "K8s is ready — install Cilium for eBPF network policy and Hubble observability."
	}
	return out
}

func (s *Service) AIInfraSummary(vectors []VectorEngineStatus, weights WeightsSummary) AIInfraSummary {
	out := AIInfraSummary{VectorDBTotal: len(vectors)}
	for _, v := range vectors {
		if v.Running {
			out.VectorDBRunning++
		}
	}
	out.ModelWeightBytes = weights.TotalBytes
	out.ModelWeightHuman = weights.TotalHuman
	if s.aihub != nil {
		hf := s.aihub.HuggingFaceStatus()
		out.HFInstalled = hf.Installed
		out.HFRunning = hf.TGIRunning || hf.OllamaRunning || hf.WebUIRunning
	}
	out.RAGReady = out.VectorDBRunning > 0 && (out.HFRunning || out.ModelWeightBytes > 0)
	if !out.HFInstalled && out.VectorDBRunning == 0 {
		out.Hint = "Install Hugging Face AI and a vector DB (Milvus/Qdrant/Weaviate) to build local RAG."
	} else if out.VectorDBRunning > 0 && !out.RAGReady {
		out.Hint = "Vector DB is running — deploy models via AI Hub to complete the RAG pipeline."
	} else if out.RAGReady {
		out.Hint = "RAG stack is partially ready: vector store + model runtime detected."
	}
	return out
}

func computeInfraHealthScore(cn CloudNativeSummary, ai AIInfraSummary, sec SecurityIntelSummary) int {
	score := 20
	if cn.DockerRunning {
		score += 10
	}
	if cn.K3sRunning || cn.K8sReady {
		score += 15
	}
	if cn.K8sReady {
		score += 10
	}
	if cn.CiliumReady {
		score += 10
	}
	if cn.MetricsRunning > 0 {
		score += 10
	}
	if ai.HFRunning {
		score += 10
	}
	if ai.VectorDBRunning > 0 {
		score += 10
	}
	if ai.RAGReady {
		score += 5
	}
	score += min(10, sec.Score/10)
	if score > 100 {
		score = 100
	}
	return score
}

func defaultQuickLinks() []QuickLink {
	return []QuickLink{
		{Key: "k8s", Title: "Kubernetes", Path: "/k8s", Icon: "Platform"},
		{Key: "cluster", Title: "Server Cluster", Path: "/cluster", Icon: "Share"},
		{Key: "docker", Title: "Docker", Path: "/docker", Icon: "Box"},
		{Key: "compose", Title: "Compose", Path: "/compose", Icon: "Grid"},
		{Key: "ai", Title: "AI Hub", Path: "/ai", Icon: "MagicStick"},
		{Key: "devops", Title: "DevOps", Path: "/devops", Icon: "Promotion"},
		{Key: "logs", Title: "Logs", Path: "/logs", Icon: "Document"},
		{Key: "protection", Title: "Cilium / Security", Path: "/protection?tab=cilium", Icon: "Histogram"},
		{Key: "software", Title: "App Store", Path: "/software", Icon: "ShoppingCart"},
	}
}

func dockerEngineUp() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "info").CombinedOutput()
	if err != nil {
		return false
	}
	return !strings.Contains(strings.ToLower(string(out)), "cannot connect")
}
