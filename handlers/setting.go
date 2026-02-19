package handlers

import (
	"net/http"

	"hidevideo/backend/config"

	"github.com/gin-gonic/gin"
)

// GetLoginProtection 获取登录保护设置
func GetLoginProtection(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"enabled": config.GetLoginProtectionEnabled(),
	})
}

// SetLoginProtection 设置登录保护
func SetLoginProtection(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	config.SetLoginProtectionEnabled(req.Enabled)
	c.JSON(http.StatusOK, gin.H{
		"message": "设置成功",
		"enabled": req.Enabled,
	})
}
