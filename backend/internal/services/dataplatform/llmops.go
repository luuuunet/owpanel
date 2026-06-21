package dataplatform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type InferenceRuntime struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Framework string `json:"framework"`
	Container string `json:"container"`
	Running   bool   `json:"running"`
	Port      int    `json:"port"`
	Endpoint  string `json:"endpoint,omitempty"`
	GPU       bool   `json:"gpu"`
	Scheduler string `json:"scheduler"` // docker | kubernetes
}

type ModelLifecycleItem struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Version    string    `json:"version,omitempty"`
	Source     string    `json:"source"`
	SizeHuman  string    `json:"size_human"`
	Status     string    `json:"status"` // cached | deployed | snapshot
	ModifiedAt time.Time `json:"modified_at,omitempty"`
}

type LLMOpsSummary struct {
	HFInstalled    bool               `json:"hf_installed"`
	HFModelID      string             `json:"hf_model_id,omitempty"`
	HFRuntime      string             `json:"hf_runtime,omitempty"`
	GPUAvailable   bool               `json:"gpu_available"`
	GPUDevices     []string           `json:"gpu_devices,omitempty"`
	Runtimes       []InferenceRuntime `json:"runtimes"`
	Models         []ModelLifecycleItem `json:"models"`
	SnapshotCount  int                `json:"snapshot_count"`
	DeployPath     string             `json:"deploy_path"`
	SyncCommand    string             `json:"sync_command"`
	Hint           string             `json:"hint,omitempty"`
}

var inferenceContainers = []struct {
	Key, Name, Framework, Container string
	Port                            int
}{
	{Key: "tgi", Name: "Text Generation Inference", Framework: "TGI", Container: "owpanel-hf-tgi", Port: 8095},
	{Key: "ollama", Name: "Ollama", Framework: "Ollama", Container: "owpanel-ollama", Port: 11434},
	{Key: "vllm", Name: "vLLM", Framework: "vLLM", Container: "owpanel-vllm", Port: 8000},
	{Key: "open-webui", Name: "Open WebUI", Framework: "Chat UI", Container: "owpanel-open-webui", Port: 8080},
}

func (s *Service) LLMOps() LLMOpsSummary {
	out := LLMOpsSummary{
		DeployPath:  "/ai",
		SyncCommand: "rsync -avz " + filepath.Join(s.dataDir, "ai/") + " user@node:/opt/owpanel/data/ai/",
	}
	if s.aihub != nil {
		gpu := s.aihub.GPUInfo()
		out.GPUAvailable = gpu.Available
		out.GPUDevices = gpu.Devices
		hf := s.aihub.HuggingFaceStatus()
		out.HFInstalled = hf.Installed
		out.HFModelID = hf.ModelID
		out.HFRuntime = hf.Runtime
	}
	for _, spec := range inferenceContainers {
		rt := InferenceRuntime{
			Key: spec.Key, Name: spec.Name, Framework: spec.Framework,
			Container: spec.Container, Port: spec.Port,
			Scheduler: "docker", GPU: out.GPUAvailable && spec.Key != "open-webui",
		}
		if dockerRunning(spec.Container) {
			rt.Running = true
			rt.Endpoint = formatEndpoint(spec.Port)
		}
		out.Runtimes = append(out.Runtimes, rt)
	}
	weights := s.listWeightAssets()
	for _, w := range weights {
		st := "cached"
		if strings.HasPrefix(w.Source, "ollama") && w.Path == "docker:owpanel-ollama" {
			st = "deployed"
		}
		out.Models = append(out.Models, ModelLifecycleItem{
			ID: w.ID, Name: w.Name, Version: w.Version, Source: w.Source,
			SizeHuman: w.SizeHuman, Status: st, ModifiedAt: w.ModifiedAt,
		})
	}
	backupDir := filepath.Join(s.dataDir, "ai", "weights-backups")
	if entries, err := os.ReadDir(backupDir); err == nil {
		out.SnapshotCount = len(entries)
	}
	runningCount := 0
	for _, r := range out.Runtimes {
		if r.Running {
			runningCount++
		}
	}
	switch {
	case !out.HFInstalled && runningCount == 0:
		out.Hint = "Deploy Hugging Face AI (TGI/Ollama) or vLLM from App Store to start LLMOps."
	case out.HFInstalled && runningCount == 0:
		out.Hint = "Models installed but inference is stopped — start TGI/Ollama/vLLM containers."
	case out.GPUAvailable && !hasRunningGPURuntime(out.Runtimes):
		out.Hint = "GPU detected — prefer vLLM or TGI with USE_GPU for higher throughput."
	default:
		out.Hint = "Use snapshots before model upgrades; rsync ai/ directory for multi-node hot sync."
	}
	return out
}

func hasRunningGPURuntime(rts []InferenceRuntime) bool {
	for _, r := range rts {
		if r.Running && r.GPU && r.Key != "open-webui" {
			return true
		}
	}
	return false
}

func formatEndpoint(port int) string {
	return "http://127.0.0.1:" + itoa(port)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func dockerRunning(name string) bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "true"
}
