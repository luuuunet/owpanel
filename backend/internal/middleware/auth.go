package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/luuuunet/owpanel/internal/auth"
)

func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

func authenticate(c *gin.Context, authSvc *auth.Service, allowQueryToken bool) bool {
	token := bearerToken(c)
	if token == "" && allowQueryToken {
		token = c.Query("token")
	}
	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization"})
		return false
	}

	claims, err := authSvc.ParseToken(token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return false
	}
	// Temp tokens issued during 2FA login must not access the panel API.
	if claims.TotpPending {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "TOTP verification required"})
		return false
	}

	role := claims.Role
	perms := claims.Permissions
	quota := claims.DiskQuotaMB
	// Refresh non-admin permissions from DB so revokes take effect before JWT expiry.
	if claims.UserID > 0 && role != "admin" {
		if u, err := authSvc.GetUser(claims.UserID); err == nil && u != nil {
			role = u.Role
			perms = u.Permissions
			quota = u.DiskQuotaMB
		}
	}

	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("role", role)
	c.Set("permissions", perms)
	c.Set("disk_quota_mb", quota)
	return true
}

// Auth validates JWT from Authorization Bearer header only.
func Auth(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticate(c, authSvc, false) {
			c.Next()
		}
	}
}

// AuthAllowQueryToken validates Bearer header or ?token= (browser WebSocket only).
func AuthAllowQueryToken(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticate(c, authSvc, true) {
			c.Next()
		}
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin required"})
			return
		}
		c.Next()
	}
}

func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		perms, _ := c.Get("permissions")
		roleStr, _ := role.(string)
		permStr, _ := perms.(string)
		if auth.CanAccess(roleStr, permStr, perm) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	}
}

func RequireShellAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		perms, _ := c.Get("permissions")
		roleStr, _ := role.(string)
		permStr, _ := perms.(string)
		switch roleStr {
		case "admin", "user":
			c.Next()
			return
		case "subuser":
			if auth.CanAccess(roleStr, permStr, "bastion") {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "shell access denied"})
	}
}
