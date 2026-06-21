package dataplatform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/luuuunet/owpanel/internal/services/appstore"
)

type VectorEngineStatus struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Installed   bool     `json:"installed"`
	Running     bool     `json:"running"`
	Status      string   `json:"status"`
	Port        int      `json:"port"`
	Endpoint    string   `json:"endpoint"`
	UseCase     string   `json:"use_case"`
	Collections []string `json:"collections,omitempty"`
	VectorCount int64    `json:"vector_count,omitempty"`
	Message     string   `json:"message,omitempty"`
}

var vectorEngines = []struct {
	Key, Name, UseCase string
	Port               int
}{
	{Key: "milvus", Name: "Milvus", UseCase: "RAG embeddings, large-scale vector search", Port: 19530},
	{Key: "qdrant", Name: "Qdrant", UseCase: "Low-latency semantic search & filtering", Port: 6333},
	{Key: "weaviate", Name: "Weaviate", UseCase: "GraphQL vector DB with hybrid search", Port: 8080},
}

func (s *Service) VectorEngines() []VectorEngineStatus {
	out := make([]VectorEngineStatus, 0, len(vectorEngines))
	for _, e := range vectorEngines {
		st := VectorEngineStatus{
			Key:      e.Key,
			Name:     e.Name,
			UseCase:  e.UseCase,
			Port:     e.Port,
			Endpoint: fmt.Sprintf("http://127.0.0.1:%d", e.Port),
		}
		if s.appstore != nil {
			app, err := s.appstore.Get(e.Key)
			if err == nil && app.Installed {
				st.Installed = true
				live := s.appstore.LiveStatus(e.Key)
				st.Status = live
				st.Running = live == "running"
			}
		}
		if st.Running {
			cols, count, msg := probeVectorCollections(e.Key, e.Port)
			st.Collections = cols
			st.VectorCount = count
			if msg != "" {
				st.Message = msg
			}
		}
		out = append(out, st)
	}
	return out
}

func probeVectorCollections(key string, port int) ([]string, int64, string) {
	client := &http.Client{Timeout: 4 * time.Second}
	switch key {
	case "qdrant":
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/collections", port))
		if err != nil {
			return nil, 0, err.Error()
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var parsed struct {
			Result struct {
				Collections []struct {
					Name string `json:"name"`
				} `json:"collections"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, 0, ""
		}
		names := make([]string, 0, len(parsed.Result.Collections))
		var total int64
		for _, c := range parsed.Result.Collections {
			names = append(names, c.Name)
			total += countQdrantVectors(client, port, c.Name)
		}
		return names, total, ""
	case "milvus":
		resp, err := client.Post(
			fmt.Sprintf("http://127.0.0.1:%d/v2/vectordb/collections/list", port),
			"application/json",
			strings.NewReader(`{}`),
		)
		if err != nil {
			// fallback legacy API on 9091
			return probeMilvusLegacy(client)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var parsed struct {
			Data []struct {
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return probeMilvusLegacy(client)
		}
		names := make([]string, 0, len(parsed.Data))
		for _, c := range parsed.Data {
			names = append(names, c.Name)
		}
		return names, 0, ""
	case "weaviate":
		resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/schema", port))
		if err != nil {
			return nil, 0, err.Error()
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		var parsed struct {
			Classes []struct {
				Class string `json:"class"`
			} `json:"classes"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, 0, ""
		}
		names := make([]string, 0, len(parsed.Classes))
		for _, c := range parsed.Classes {
			names = append(names, c.Class)
		}
		return names, 0, ""
	}
	return nil, 0, ""
}

func probeMilvusLegacy(client *http.Client) ([]string, int64, string) {
	resp, err := client.Get("http://127.0.0.1:9091/api/v1/collections")
	if err != nil {
		return nil, 0, err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		CollectionNames []string `json:"collection_names"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, 0, ""
	}
	return parsed.CollectionNames, 0, ""
}

func countQdrantVectors(client *http.Client, port int, name string) int64 {
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/collections/%s", port, name))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Result struct {
			PointsCount int64 `json:"points_count"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0
	}
	return parsed.Result.PointsCount
}

// ComposeAppKeys lists store apps installed via dataplatform compose stacks.
var ComposeAppKeys = map[string]bool{
	"milvus": true, "weaviate": true, "victoria-metrics": true, "ceph": true,
}

func ComposeInstalled(dataDir, key string) bool {
	return appstore.DataPlatformComposeInstalled(dataDir, key)
}

func ComposeRunning(dataDir, key string) bool {
	return appstore.DataPlatformComposeRunning(dataDir, key)
}
