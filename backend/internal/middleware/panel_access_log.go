package middleware

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type accessRecorder func(ip, method, path string, status int, bytes uint64, referer, userAgent string)

// PanelAccessLog writes nginx-compatible access lines for traffic map ingestion.
func PanelAccessLog(dataDir string, record accessRecorder) gin.HandlerFunc {
	logDir := filepath.Join(dataDir, "logs")
	_ = os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "panel_access.log")

	var mu sync.Mutex

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.Request.URL.Path
		if shouldSkipTrafficPath(path) {
			return
		}

		status := c.Writer.Status()
		size := max(c.Writer.Size(), 0)
		ip := clientIP(c)
		method := c.Request.Method
		proto := c.Request.Proto
		if proto == "" {
			proto = "HTTP/1.1"
		}
		referer := c.Request.Referer()
		ua := c.Request.UserAgent()
		when := start.Format("02/Jan/2006:15:04:05 -0700")
		line := fmt.Sprintf(`%s - - [%s] "%s %s %s" %d %d "%s" "%s"`,
			ip, when, method, path, proto, status, size, referer, ua)

		mu.Lock()
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, _ = f.WriteString(line + "\n")
			_ = f.Close()
		}
		mu.Unlock()

		if record != nil {
			record(ip, method, path, status, uint64(size), referer, ua)
		}
	}
}

func shouldSkipTrafficPath(path string) bool {
	if path == "" || path == "/favicon.ico" {
		return true
	}
	if strings.HasPrefix(path, "/assets/") || strings.HasPrefix(path, "/geo/") {
		return true
	}
	if strings.HasPrefix(path, "/api/analytics/traffic-map") {
		return true
	}
	return false
}

func clientIP(c *gin.Context) string {
	if xff := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return xff
	}
	if xri := strings.TrimSpace(c.GetHeader("X-Real-IP")); xri != "" {
		return xri
	}
	return c.ClientIP()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
