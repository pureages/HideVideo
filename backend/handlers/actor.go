package handlers

import (
	"net/http"
	"strconv"

	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-gonic/gin"
)

// GetActors 获取所有演员
func GetActors(c *gin.Context) {
	var actors []models.Actor
	if err := database.DB.Order("sort_order ASC, id ASC").Find(&actors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取演员列表失败"})
		return
	}
	c.JSON(http.StatusOK, actors)
}

// AddActor 添加演员
func AddActor(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入演员名称"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "演员名称不能为空"})
		return
	}

	// 检查是否已存在
	var existing models.Actor
	if err := database.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "演员已存在"})
		return
	}

	actor := models.Actor{Name: req.Name}
	if err := database.DB.Create(&actor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加演员失败"})
		return
	}

	c.JSON(http.StatusOK, actor)
}

// UpdateActor 更新演员
func UpdateActor(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入演员名称"})
		return
	}

	var actor models.Actor
	if err := database.DB.First(&actor, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "演员不存在"})
		return
	}

	// 检查名称是否与其他演员重复
	var existing models.Actor
	if err := database.DB.Where("name = ? AND id != ?", req.Name, id).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "演员名称已存在"})
		return
	}

	actor.Name = req.Name
	database.DB.Save(&actor)

	c.JSON(http.StatusOK, actor)
}

// DeleteActor 删除演员
func DeleteActor(c *gin.Context) {
	id := c.Param("id")

	// 删除关联
	database.DB.Where("actor_id = ?", id).Delete(&models.VideoActor{})

	if err := database.DB.Delete(&models.Actor{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除演员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ReorderActors 批量更新演员排序
func ReorderActors(c *gin.Context) {
	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	for i, id := range req.IDs {
		database.DB.Model(&models.Actor{}).Where("id = ?", id).Update("sort_order", i)
	}

	c.JSON(http.StatusOK, gin.H{"message": "排序已更新"})
}

// GetVideoActors 获取视频的演员列表
func GetVideoActors(c *gin.Context) {
	videoID := c.Param("id")

	var actors []models.Actor
	if err := database.DB.Model(&models.Actor{}).
		Joins("JOIN video_actors ON video_actors.actor_id = actors.id").
		Where("video_actors.video_id = ?", videoID).
		Find(&actors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取演员列表失败"})
		return
	}

	c.JSON(http.StatusOK, actors)
}

// AddVideoActor 添加视频演员
func AddVideoActor(c *gin.Context) {
	videoID := c.Param("id")
	var req struct {
		ActorID uint `json:"actor_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请提供演员ID"})
		return
	}

	// 检查视频是否存在
	var video models.Video
	if err := database.DB.First(&video, videoID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 检查演员是否存在
	var actor models.Actor
	if err := database.DB.First(&actor, req.ActorID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "演员不存在"})
		return
	}

	// 检查是否已关联
	var existing models.VideoActor
	if err := database.DB.Where("video_id = ? AND actor_id = ?", videoID, req.ActorID).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "演员已添加"})
		return
	}

	// 添加关联
	videoActor := models.VideoActor{VideoID: video.ID, ActorID: req.ActorID}
	database.DB.Create(&videoActor)

	c.JSON(http.StatusOK, actor)
}

// RemoveVideoActor 删除视频演员
func RemoveVideoActor(c *gin.Context) {
	videoID := c.Param("id")
	actorID := c.Param("actorId")

	if err := database.DB.Where("video_id = ? AND actor_id = ?", videoID, actorID).Delete(&models.VideoActor{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除演员失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// GetActorVideos 获取演员参演的所有视频
func GetActorVideos(c *gin.Context) {
	actorID := c.Param("id")

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var videos []models.Video
	var total int64

	// 查询该演员的所有视频
	subQuery := database.DB.Model(&models.VideoActor{}).
		Select("video_id").
		Where("actor_id = ?", actorID)

	database.DB.Model(&models.Video{}).
		Where("id IN (?)", subQuery).
		Count(&total)

	offset := (page - 1) * pageSize
	database.DB.Preload("Tags").
		Where("id IN (?)", subQuery).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&videos)

	// 处理封面路径
	for i := range videos {
		if videos[i].CoverPath != "" {
			// 提取文件名
			lastSlash := -1
			for j := len(videos[i].CoverPath) - 1; j >= 0; j-- {
				if videos[i].CoverPath[j] == '/' || videos[i].CoverPath[j] == '\\' {
					lastSlash = j
					break
				}
			}
			if lastSlash >= 0 {
				videos[i].CoverPath = "/covers/" + videos[i].CoverPath[lastSlash+1:]
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"list":        videos,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (int(total) + pageSize - 1) / pageSize,
	})
}
