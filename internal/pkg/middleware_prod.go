//go:build !dev

package pkg

import "github.com/gin-gonic/gin"

func applyDevAdminBypass(c *gin.Context) bool {
	return false
}
