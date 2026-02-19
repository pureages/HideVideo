package handlers

import (
	"net/http"
	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-gonic/gin"
)

// GetComments 获取视频评论
func GetComments(c *gin.Context) {
	videoID := c.Param("id")

	var comments []models.Comment
	if err := database.DB.Where("video_id = ?", videoID).
		Preload("User").
		Order("created_at DESC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论失败"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// AddComment 添加评论
func AddComment(c *gin.Context) {
	videoID := c.Param("id")
	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入评论内容"})
		return
	}

	// 检查视频是否存在
	var video models.Video
	if err := database.DB.First(&video, videoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 获取当前用户ID
	userID := c.GetUint("user_id")

	comment := models.Comment{
		VideoID: video.ID,
		UserID:   userID,
		Content:  req.Content,
	}

	if err := database.DB.Create(&comment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加评论失败"})
		return
	}

	// 重新查询以获取用户信息
	database.DB.Preload("User").First(&comment, comment.ID)

	c.JSON(http.StatusOK, gin.H{"message": "评论成功", "comment": comment})
}

// DeleteComment 删除评论
func DeleteComment(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Comment{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除评论失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
