//go:build dev

package pkg

import (
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func init() {
	if strings.ToLower(strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS"))) != "" {
		slog.Warn("⚠️  DEV ADMIN BYPASS 环境变量已设置 — 仅限开发环境，切勿在生产中使用！")
	}
}

func applyDevAdminBypass(c *gin.Context) bool {
	if !devAdminBypassEnabled() || !isLoopbackRequest(c) {
		return false
	}

	c.Set("user_id", uint(0))
	c.Set("username", "local-dev-admin")
	c.Set("role", "admin")
	c.Set("dev_admin_bypass", true)
	return true
}

func devAdminBypassEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func isLoopbackRequest(c *gin.Context) bool {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
