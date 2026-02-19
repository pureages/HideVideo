package handlers

import (
	"net/http"
	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-gonic/gin"
)

// GetTags 获取标签列表
func GetTags(c *gin.Context) {
	var tags []models.Tag
	// 按sort_order排序，相同则按name排序
	if err := database.DB.Order("sort_order ASC, name ASC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签列表失败"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// AddTag 添加标签
func AddTag(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入标签名称"})
		return
	}

	// 检查标签是否已存在
	var count int64
	database.DB.Model(&models.Tag{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签已存在"})
		return
	}

	tag := models.Tag{Name: req.Name}
	if err := database.DB.Create(&tag).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "添加成功", "tag": tag})
}

// DeleteTag 删除标签
func DeleteTag(c *gin.Context) {
	id := c.Param("id")

	// 删除标签与视频的关联
	database.DB.Where("tag_id = ?", id).Delete(&models.VideoTag{})

	// 删除标签
	if err := database.DB.Delete(&models.Tag{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// UpdateTag 更新标签
func UpdateTag(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入标签名称"})
		return
	}

	// 检查标签是否已存在（排除自己）
	var count int64
	database.DB.Model(&models.Tag{}).Where("name = ? AND id != ?", req.Name, id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签名称已存在"})
		return
	}

	// 更新标签
	if err := database.DB.Model(&models.Tag{}).Where("id = ?", id).Update("name", req.Name).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新标签失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// AddVideoTag 为视频添加标签
func AddVideoTag(c *gin.Context) {
	videoID := c.Param("id")
	var req struct {
		TagID uint `json:"tag_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择标签"})
		return
	}

	// 检查视频是否存在
	var video models.Video
	if err := database.DB.First(&video, videoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 检查标签是否存在
	var tag models.Tag
	if err := database.DB.First(&tag, req.TagID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "标签不存在"})
		return
	}

	// 检查关联是否已存在
	var count int64
	database.DB.Model(&models.VideoTag{}).Where("video_id = ? AND tag_id = ?", videoID, req.TagID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标签已添加"})
		return
	}

	// 添加关联
	videoTag := models.VideoTag{VideoID: video.ID, TagID: tag.ID}
	database.DB.Create(&videoTag)

	c.JSON(http.StatusOK, gin.H{"message": "添加标签成功"})
}

// RemoveVideoTag 移除视频标签
func RemoveVideoTag(c *gin.Context) {
	videoID := c.Param("id")
	tagID := c.Param("tagId")

	database.DB.Where("video_id = ? AND tag_id = ?", videoID, tagID).Delete(&models.VideoTag{})

	c.JSON(http.StatusOK, gin.H{"message": "移除标签成功"})
}

// GetVideoTags 获取视频的标签
func GetVideoTags(c *gin.Context) {
	videoID := c.Param("id")

	var tags []models.Tag
	if err := database.DB.Joins("JOIN video_tags ON video_tags.tag_id = tags.id").
		Where("video_tags.video_id = ?", videoID).
		Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取标签失败"})
		return
	}

	c.JSON(http.StatusOK, tags)
}

// ReorderTags 批量更新标签排序
func ReorderTags(c *gin.Context) {
	var req struct {
		TagIDs []uint `json:"tag_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供标签ID列表"})
		return
	}

	// 批量更新排序
	for i, tagID := range req.TagIDs {
		if err := database.DB.Model(&models.Tag{}).Where("id = ?", tagID).Update("sort_order", i).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新排序失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序更新成功"})
}
