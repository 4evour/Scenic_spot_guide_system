//go:build dev

package pkg

import (
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// IsDevBuild 标识当前二进制是否以 -tags dev 编译。
// main.go 据此在 release 模式下拒绝启动 dev 构建,防止误部署带后门的二进制。
const IsDevBuild = true

func init() {
	if devAdminBypassEnabled() {
		slog.Warn("⚠️  DEV ADMIN BYPASS 已启用 — 仅限本地开发，切勿在生产中使用！",
			"hint", "需同时设置 SCENIC_GUIDE_DEV_ADMIN_BYPASS 与 SCENIC_GUIDE_DEV_ALLOW_BYPASS 才会生效")
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

// devAdminBypassEnabled 要求两个环境变量同时为真才会启用旁路:
//   - SCENIC_GUIDE_DEV_ADMIN_BYPASS: 主开关
//   - SCENIC_GUIDE_DEV_ALLOW_BYPASS: 确认开关(防止运维误设单个变量即触发)
//
// 单一变量被误设不会激活旁路,降低误用风险。
func devAdminBypassEnabled() bool {
	main := strings.ToLower(strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEV_ADMIN_BYPASS")))
	confirm := strings.ToLower(strings.TrimSpace(os.Getenv("SCENIC_GUIDE_DEV_ALLOW_BYPASS")))
	isTrue := func(v string) bool {
		return v == "1" || v == "true" || v == "yes" || v == "on"
	}
	return isTrue(main) && isTrue(confirm)
}

func isLoopbackRequest(c *gin.Context) bool {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
