package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"hidevideo/backend/config"
	"hidevideo/backend/database"
	"hidevideo/backend/models"
	"hidevideo/backend/utils"

	"github.com/gin-gonic/gin"
)

// GetLibraries 获取视频库列表
func GetLibraries(c *gin.Context) {
	var libraries []models.VideoLibrary
	result := database.DB.Order("created_at DESC").Find(&libraries)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频库列表失败"})
		return
	}
	c.JSON(http.StatusOK, libraries)
}

// AddLibrary 添加视频库
func AddLibrary(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Path string `json:"path"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入库名称"})
		return
	}

	library := models.VideoLibrary{
		Name: req.Name,
		Path: req.Path,
	}

	// 检查是否有软删除的同名库，如果有则恢复
	var deletedLib models.VideoLibrary
	if err := database.DB.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).First(&deletedLib).Error; err == nil {
		// 恢复软删除的记录
		database.DB.Model(&deletedLib).Unscoped().Updates(map[string]interface{}{
			"deleted_at": nil,
			"path":       req.Path,
		})
		c.JSON(http.StatusOK, gin.H{
			"message": "恢复成功",
			"library": deletedLib,
		})
		return
	}

	// 验证路径是否存在
	if req.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入路径"})
		return
	}

	// 检查路径是否存在
	if !utils.IsDir(req.Path) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "路径不存在或不是有效目录"})
		return
	}

	// 检查是否有软删除的同路径库，如果有则恢复
	var deletedPathLib models.VideoLibrary
	if err := database.DB.Unscoped().Where("path = ? AND deleted_at IS NOT NULL", req.Path).First(&deletedPathLib).Error; err == nil {
		// 恢复软删除的记录，同时更新名称
		database.DB.Model(&deletedPathLib).Unscoped().Updates(map[string]interface{}{
			"deleted_at": nil,
			"name":       req.Name,
		})
		// 重新查询获取更新后的数据
		database.DB.First(&deletedPathLib, deletedPathLib.ID)
		c.JSON(http.StatusOK, gin.H{
			"message": "恢复成功",
			"library": deletedPathLib,
		})
		return
	}

	// 检查当前是否有同名路径的未删除库
	var existingLib models.VideoLibrary
	if err := database.DB.Where("path = ?", req.Path).First(&existingLib).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该路径已添加"})
		return
	}

	library.Path = req.Path

	result := database.DB.Create(&library)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加视频库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "添加成功",
		"library": library,
	})
}

// DeleteLibrary 删除视频库
func DeleteLibrary(c *gin.Context) {
	id := c.Param("id")
	var library models.VideoLibrary

	if err := database.DB.First(&library, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	// 删除视频库的封面文件
	var videos []models.Video
	database.DB.Where("library_id = ?", id).Find(&videos)
	for _, video := range videos {
		if video.CoverPath != "" {
			utils.FileExists(video.CoverPath)
		}
	}

	// 删除视频库（级联删除视频、评论、标签关联）
	// 先删除视频标签关联
	database.DB.Where("video_id IN ?", func() []uint {
		var ids []uint
		for _, v := range videos {
			ids = append(ids, v.ID)
		}
		return ids
	}()).Delete(&models.VideoTag{})

	// 删除评论
	database.DB.Where("video_id IN ?", func() []uint {
		var ids []uint
		for _, v := range videos {
			ids = append(ids, v.ID)
		}
		return ids
	}()).Delete(&models.Comment{})

	// 删除视频
	database.DB.Where("library_id = ?", id).Delete(&models.Video{})

	// 删除视频库
	database.DB.Delete(&library)

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ScanLibrary 扫描视频库
func ScanLibrary(c *gin.Context) {
	id := c.Param("id")
	var library models.VideoLibrary

	if err := database.DB.First(&library, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	var videos []string
	var err error

	// 本地库扫描
	videos, err = utils.GetVideoFiles(library.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("扫描失败: %v", err)})
		return
	}

	var addedCount int
	var skipCount int

	for _, videoPath := range videos {
		// 检查视频是否已存在
		var count int64
		database.DB.Model(&models.Video{}).Where("filepath = ?", videoPath).Count(&count)
		if count > 0 {
			skipCount++
			continue
		}

		var duration float64
		var width, height int
		var codec string

		// 获取视频信息
		videoInfo, err := utils.GetVideoInfo(videoPath)
		if err != nil {
			continue
		}
		duration = videoInfo.Duration
		width = videoInfo.Width
		height = videoInfo.Height
		codec = videoInfo.Codec

		video := models.Video{
			LibraryID: library.ID,
			Filename:  filepath.Base(videoPath),
			Filepath:  videoPath,
			Duration:  duration,
			Width:     width,
			Height:    height,
			Codec:     codec,
		}

		if err := database.DB.Create(&video).Error; err != nil {
			continue
		}
		addedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "扫描完成",
		"added":       addedCount,
		"skipped":     skipCount,
		"total_found": len(videos),
	})
}

// GenerateCovers 生成视频封面
func GenerateCovers(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Second float64 `json:"second" binding:"required"`
		Mode   string  `json:"mode"` // "new" 仅新视频, "reset" 全部重置
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入截图秒数"})
		return
	}

	// 默认模式
	if req.Mode == "" {
		req.Mode = "reset"
	}

	var library models.VideoLibrary
	if err := database.DB.First(&library, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	var videos []models.Video
	if req.Mode == "new" {
		// 仅生成没有封面的视频（检查cover_path是否为空或NULL）
		database.DB.Where("library_id = ? AND (cover_path IS NULL OR cover_path = '' OR LENGTH(cover_path) = 0)", id).Find(&videos)
	} else {
		// 全部重置
		database.DB.Where("library_id = ?", id).Find(&videos)
	}

	var successCount int
	var failCount int

	for _, video := range videos {
		coverPath, err := utils.GenerateCover(video.Filepath, video.ID, req.Second)
		if err != nil {
			failCount++
			continue
		}

		// 更新视频的封面路径（使用相对路径）
		relativePath := coverPath
		database.DB.Model(&video).Update("cover_path", relativePath)
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "封面生成完成",
		"success": successCount,
		"failed":  failCount,
		"total":   len(videos),
	})
}

// CleanInvalidIndex 清除错误索引
func CleanInvalidIndex(c *gin.Context) {
	var videos []models.Video
	database.DB.Find(&videos)

	var deletedVideos int
	var deletedCovers int
	var deletedOrphanCovers int
	var deletedLibraries int

	// 遍历所有视频，检查文件是否存在
	for _, video := range videos {
		// 检查视频文件是否存在
		if _, err := os.Stat(video.Filepath); os.IsNotExist(err) {
			// 删除视频相关的评论
			database.DB.Where("video_id = ?", video.ID).Delete(&models.Comment{})
			// 删除视频标签关联
			database.DB.Where("video_id = ?", video.ID).Delete(&models.VideoTag{})
			// 删除视频
			database.DB.Delete(&video)
			deletedVideos++
			continue
		}

		// 检查封面是否存在
		if video.CoverPath != "" {
			if _, err := os.Stat(video.CoverPath); os.IsNotExist(err) {
				// 删除封面路径
				database.DB.Model(&video).Update("cover_path", "")
				deletedCovers++
			}
		}
	}

	// 查询数据库中所有视频的封面路径
	var coverPaths []string
	database.DB.Model(&models.Video{}).Where("cover_path != ?", "").Pluck("cover_path", &coverPaths)

	// 创建map方便快速查找
	coverPathMap := make(map[string]bool)
	for _, path := range coverPaths {
		coverPathMap[path] = true
	}

	// 读取封面目录下的所有文件
	coverDir := config.ServerConfig.StaticPath
	entries, err := os.ReadDir(coverDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			// 获取文件完整路径
			coverFilePath := filepath.Join(coverDir, entry.Name())
			// 检查是否在数据库中有引用
			if !coverPathMap[coverFilePath] {
				// 删除多余封面
				os.Remove(coverFilePath)
				deletedOrphanCovers++
			}
		}
	}

	// 清理已删除的视频库（软删除的库）
	// 使用 Unscoped 查询包括软删除在内的所有记录
	var deletedLibs []models.VideoLibrary
	database.DB.Unscoped().Where("deleted_at IS NOT NULL").Find(&deletedLibs)

	// 检查每个软删除的库是否还有关联的视频
	for _, lib := range deletedLibs {
		var videoCount int64
		database.DB.Unscoped().Model(&models.Video{}).Where("library_id = ?", lib.ID).Count(&videoCount)
		if videoCount == 0 {
			// 彻底删除该视频库记录
			database.DB.Unscoped().Delete(&lib)
			deletedLibraries++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                "清理完成",
		"deleted_videos":         deletedVideos,
		"deleted_covers":         deletedCovers,
		"deleted_orphan_covers":  deletedOrphanCovers,
		"deleted_libraries":      deletedLibraries,
	})
}
