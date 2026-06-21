package dataplatform

type KnowledgeBaseApp struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Port      int    `json:"port"`
	UseCase   string `json:"use_case"`
}

type EmbeddingStats struct {
	Engine         string `json:"engine"`
	Collections    int    `json:"collections"`
	VectorCount    int64  `json:"vector_count"`
	StorageLinked  bool   `json:"storage_linked"`
}

type DataOpsSummary struct {
	VectorDBs      []VectorEngineStatus `json:"vector_dbs"`
	StorageMeta    []StorageEngineMeta  `json:"storage_meta"`
	KnowledgeApps  []KnowledgeBaseApp   `json:"knowledge_apps"`
	EmbeddingStats []EmbeddingStats     `json:"embedding_stats"`
	RAGReady       bool                 `json:"rag_ready"`
	SyncHint       string               `json:"sync_hint"`
	Hint           string               `json:"hint,omitempty"`
}

var knowledgeApps = []struct {
	Key, Name, UseCase string
	Port               int
}{
	{Key: "dify", Name: "Dify", UseCase: "RAG workflow & knowledge base", Port: 8091},
	{Key: "flowise", Name: "Flowise", UseCase: "LangChain visual RAG builder", Port: 3010},
	{Key: "fastgpt", Name: "FastGPT", UseCase: "Knowledge QA platform", Port: 3002},
	{Key: "anythingllm", Name: "AnythingLLM", UseCase: "Document RAG workspace", Port: 3001},
	{Key: "maxkb", Name: "MaxKB", UseCase: "Enterprise knowledge base", Port: 8080},
}

func (s *Service) DataOps() DataOpsSummary {
	vectors := s.VectorEngines()
	storage := s.StorageMetadata()
	var apps []KnowledgeBaseApp
	for _, ka := range knowledgeApps {
		item := KnowledgeBaseApp{Key: ka.Key, Name: ka.Name, UseCase: ka.UseCase, Port: ka.Port}
		if s.appstore != nil {
			if app, err := s.appstore.Get(ka.Key); err == nil && app.Installed {
				item.Installed = true
				item.Running = s.appstore.LiveStatus(ka.Key) == "running"
			}
		}
		apps = append(apps, item)
	}
	var embedStats []EmbeddingStats
	storageRunning := false
	for _, st := range storage {
		if st.Running {
			storageRunning = true
		}
	}
	for _, v := range vectors {
		es := EmbeddingStats{
			Engine:        v.Name,
			Collections:   len(v.Collections),
			VectorCount:   v.VectorCount,
			StorageLinked: storageRunning,
		}
		embedStats = append(embedStats, es)
	}
	ragReady := false
	runningVec := 0
	for _, v := range vectors {
		if v.Running {
			runningVec++
			ragReady = true
		}
	}
	for _, a := range apps {
		if a.Running {
			ragReady = true
		}
	}
	out := DataOpsSummary{
		VectorDBs:      vectors,
		StorageMeta:    storage,
		KnowledgeApps:  apps,
		EmbeddingStats: embedStats,
		RAGReady:       ragReady,
		SyncHint:       "Point vector DB + MinIO/Ceph endpoints to the same cluster for distributed embeddings backup.",
	}
	switch {
	case runningVec == 0:
		out.Hint = "Install Milvus, Qdrant, or Weaviate via App Store for embedding storage."
	case !storageRunning:
		out.Hint = "Vector DB running — add MinIO/Ceph for durable embedding & document object storage."
	default:
		out.Hint = "Embeddings pipeline ready: vector store + object storage detected."
	}
	return out
}
