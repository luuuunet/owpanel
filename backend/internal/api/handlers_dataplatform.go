package api

import (
	"github.com/gin-gonic/gin"
	"github.com/luuuunet/owpanel/internal/api/response"
)

func (s *Server) registerDataPlatformRoutes(g *gin.RouterGroup) {
	s.registerInfraHubRoutesOnPrefix(g, "/data-platform")
	s.registerInfraHubRoutesOnPrefix(g, "/infra-hub")
}

func (s *Server) registerInfraHubRoutesOnPrefix(g *gin.RouterGroup, prefix string) {
	g.GET(prefix+"/overview", s.handleDataPlatformOverview)
	g.GET(prefix+"/vector", s.handleDataPlatformVector)
	g.GET(prefix+"/metrics", s.handleDataPlatformMetrics)
	g.GET(prefix+"/weights", s.handleDataPlatformWeights)
	g.POST(prefix+"/weights/snapshot", s.handleDataPlatformWeightSnapshot)
	g.DELETE(prefix+"/weights", s.handleDataPlatformWeightDelete)
	g.GET(prefix+"/security", s.handleDataPlatformSecurity)
	g.GET(prefix+"/orchestration", s.handleDataPlatformOrchestration)
	g.GET(prefix+"/llmops", s.handleDataPlatformLLMOps)
	g.GET(prefix+"/dataops", s.handleDataPlatformDataOps)
	g.GET(prefix+"/aiops", s.handleDataPlatformAIOps)
	g.GET(prefix+"/secops", s.handleDataPlatformSecOps)
	g.GET(prefix+"/storage", s.handleDataPlatformStorage)
}

func (s *Server) handleDataPlatformOverview(c *gin.Context) {
	response.OK(c, s.dataplatform.Overview())
}

func (s *Server) handleDataPlatformVector(c *gin.Context) {
	response.OK(c, s.dataplatform.VectorEngines())
}

func (s *Server) handleDataPlatformMetrics(c *gin.Context) {
	response.OK(c, s.dataplatform.MetricsEngines())
}

func (s *Server) handleDataPlatformWeights(c *gin.Context) {
	response.OK(c, s.dataplatform.WeightsSummary())
}

func (s *Server) handleDataPlatformWeightSnapshot(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		response.Error(c, 400, "id required")
		return
	}
	path, err := s.dataplatform.SnapshotWeight(req.ID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.OK(c, gin.H{"path": path, "message": "snapshot created"})
}

func (s *Server) handleDataPlatformWeightDelete(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		response.Error(c, 400, "id required")
		return
	}
	if err := s.dataplatform.DeleteWeightCache(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.Message(c, "deleted")
}

func (s *Server) handleDataPlatformSecurity(c *gin.Context) {
	response.OK(c, s.dataplatform.SecurityIntel())
}

func (s *Server) handleDataPlatformStorage(c *gin.Context) {
	response.OK(c, s.dataplatform.StorageMetadata())
}

func (s *Server) handleDataPlatformLLMOps(c *gin.Context) {
	response.OK(c, s.dataplatform.LLMOps())
}

func (s *Server) handleDataPlatformDataOps(c *gin.Context) {
	response.OK(c, s.dataplatform.DataOps())
}

func (s *Server) handleDataPlatformAIOps(c *gin.Context) {
	response.OK(c, s.dataplatform.AIOps())
}

func (s *Server) handleDataPlatformSecOps(c *gin.Context) {
	response.OK(c, s.dataplatform.SecOps())
}

func (s *Server) handleDataPlatformOrchestration(c *gin.Context) {
	response.OK(c, s.dataplatform.Orchestration())
}
