package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BlockWithoutSafePath rejects requests that do not use the configured security entrance prefix.
func BlockWithoutSafePath(safePath string) gin.HandlerFunc {
	sp := strings.Trim(strings.TrimSpace(safePath), "/")
	prefix := "/" + sp
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			c.Set("panel_safe_path", sp)
			c.Next()
			return
		}
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			target := prefix + "/login"
			c.Redirect(http.StatusFound, target)
			c.Abort()
			return
		}
		c.String(http.StatusNotFound, "404 — use panel security entrance /%s", sp)
		c.Abort()
	}
}
