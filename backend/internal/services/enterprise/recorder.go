package enterprise

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

type AuditRecorder struct {
	svc *Service
}

func (s *Service) Recorder() *AuditRecorder {
	return &AuditRecorder{svc: s}
}

func (r *AuditRecorder) FromGin(c *gin.Context, category, action, resource, detail, level string, success bool) {
	if r == nil || r.svc == nil {
		return
	}
	uid := c.GetUint("user_id")
	username, _ := c.Get("username")
	uname, _ := username.(string)
	_ = r.svc.Record(context.Background(), AuditRecordInput{
		UserID: uid, Username: uname, IP: c.ClientIP(), UserAgent: c.GetHeader("User-Agent"),
		Category: category, Action: action, Resource: resource, Detail: detail,
		Level: level, Success: success,
	})
}

func (r *AuditRecorder) Login(username, ip, userAgent string, success bool, reason string) {
	if r == nil || r.svc == nil {
		return
	}
	level := "info"
	if !success {
		level = "warn"
	}
	_ = r.svc.Record(context.Background(), AuditRecordInput{
		Username: username, IP: ip, UserAgent: userAgent,
		Category: "security", Action: "login", Resource: username,
		Detail: reason, Level: level, Success: success,
	})
}

func (r *AuditRecorder) SystemMutation(c *gin.Context, action, resource string, success bool, detail string) {
	r.FromGin(c, "system", action, resource, detail, "info", success)
}

func (r *AuditRecorder) InfraMutation(c *gin.Context, action, resource string, success bool, detail string) {
	level := "info"
	if !success {
		level = "warn"
	}
	r.FromGin(c, "infra", action, resource, detail, level, success)
}

func InfraAuditMiddleware(svc *Service) gin.HandlerFunc {
	rec := svc.Recorder()
	return func(c *gin.Context) {
		c.Next()
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return
		}
		path := c.Request.URL.Path
		if SkipAuditPath(path) {
			return
		}
		success := c.Writer.Status() < 400
		action, resource := parseInfraAudit(path, method)
		rec.InfraMutation(c, action, resource, success, fmt.Sprintf("status=%d", c.Writer.Status()))
	}
}

func parseInfraAudit(path, method string) (action, resource string) {
	path = strings.ToLower(path)
	resource = path
	action = strings.ToLower(method) + "_api"
	switch {
	case strings.Contains(path, "/docker/containers/") && strings.HasSuffix(path, "/restart"):
		action = "docker_restart"
	case strings.Contains(path, "/docker/containers/") && strings.HasSuffix(path, "/stop"):
		action = "docker_stop"
	case strings.Contains(path, "/docker/containers/") && strings.HasSuffix(path, "/start"):
		action = "docker_start"
	case strings.Contains(path, "/docker/containers/") && strings.Contains(path, "/recreate"):
		action = "docker_recreate"
	case strings.Contains(path, "/docker/containers/run"):
		action = "docker_run"
	case strings.Contains(path, "/docker/images/pull"):
		action = "docker_pull_image"
	case strings.Contains(path, "/docker/images/") && method == "DELETE":
		action = "docker_remove_image"
	case strings.Contains(path, "/docker/volumes"):
		action = "docker_volume_" + strings.ToLower(method)
	case strings.Contains(path, "/docker/networks"):
		action = "docker_network_" + strings.ToLower(method)
	case strings.Contains(path, "/mail/") && strings.Contains(path, "restart"):
		action = "mail_restart"
	case strings.Contains(path, "/nginx/") || strings.Contains(path, "/php/"):
		action = "webserver_" + strings.ToLower(method)
	case strings.Contains(path, "/cron/") && strings.Contains(path, "/run"):
		action = "cron_run"
	case strings.Contains(path, "/cron/reload"):
		action = "cron_reload"
	case strings.Contains(path, "/compose/"):
		action = "compose_" + strings.ToLower(method)
	}
	return action, resource
}

func SkipAuditPath(path string) bool {
	path = strings.ToLower(path)
	skip := []string{"/enterprise/audit", "/dashboard/", "/analytics/", "/auth/me"}
	for _, p := range skip {
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

func AuditMiddleware(svc *Service) gin.HandlerFunc {
	rec := svc.Recorder()
	return func(c *gin.Context) {
		c.Next()
		method := c.Request.Method
		if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
			return
		}
		path := c.Request.URL.Path
		if SkipAuditPath(path) {
			return
		}
		if strings.Contains(path, "/auth/login") || strings.Contains(path, "/cluster/nodes") ||
			strings.Contains(path, "/users") || strings.Contains(path, "/settings") ||
			strings.Contains(path, "/security/panel-access") || strings.Contains(path, "/dashboard/performance") ||
			strings.Contains(path, "/settings/migration") {
			return
		}
		success := c.Writer.Status() < 400
		level := "info"
		if !success {
			level = "warn"
		}
		action := strings.ToLower(method) + "_api"
		rec.FromGin(c, "system", action, path, fmt.Sprintf("status=%d", c.Writer.Status()), level, success)
	}
}
