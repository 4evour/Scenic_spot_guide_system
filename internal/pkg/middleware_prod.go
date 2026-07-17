//go:build !dev

package pkg

import "github.com/gin-gonic/gin"

// IsDevBuild 标识当前二进制是否以 -tags dev 编译(prod 构建恒为 false)。
const IsDevBuild = false

func applyDevAdminBypass(c *gin.Context) bool {
	return false
}
