package api

import (
	"github.com/gin-gonic/gin"
	"github.com/luuuunet/owpanel/internal/api/response"
	"github.com/luuuunet/owpanel/internal/services/posthog"
)

func (s *Server) registerPosthogRoutes(authorized *gin.RouterGroup) {
	authorized.GET("/posthog/status", s.handlePosthogStatus)
	authorized.GET("/posthog/tracking-snippet", s.handlePosthogTrackingSnippet)
	authorized.GET("/posthog/features", s.handlePosthogFeatures)
}

func (s *Server) handlePosthogStatus(c *gin.Context) {
	response.OK(c, s.posthog.Status())
}

func (s *Server) handlePosthogTrackingSnippet(c *gin.Context) {
	apiKey := c.Query("project_api_key")
	if apiKey == "" {
		apiKey = c.Query("client_id")
	}
	apiHost := c.Query("api_host")
	if apiHost == "" {
		apiHost = c.Query("api_url")
	}
	response.OK(c, s.posthog.TrackingSnippet(apiKey, apiHost))
}

func (s *Server) handlePosthogFeatures(c *gin.Context) {
	response.OK(c, gin.H{"features": posthog.DefaultFeatures()})
}
