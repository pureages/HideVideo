package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-gonic/gin"
)

// GenerateIcon 生成视频图标（小、中两种尺寸）
func GenerateIcon(c *gin.Context) {
	libraryID := c.Param("id")

	// 获取视频库
	var library models.VideoLibrary
	if err := database.DB.First(&library, libraryID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	// 获取该视频库下的所有视频
	var videos []models.Video
	database.DB.Where("library_id = ?", libraryID).Find(&videos)

	// 确保图标目录存在
	iconDir := "./backend/data/icon"
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建图标目录失败"})
		return
	}

	// 查找 ffmpeg
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到 ffmpeg"})
		return
	}

	var successCount int
	var failCount int

	for _, video := range videos {
		// 检查是否有封面
		if video.CoverPath == "" {
			failCount++
			continue
		}

		// 检查封面文件是否存在
		if _, err := os.Stat(video.CoverPath); os.IsNotExist(err) {
			failCount++
			continue
		}

		// 生成小图标（48x48）
		smallIconPath := filepath.Join(iconDir, fmt.Sprintf("icon_%d_small.png", video.ID))
		smallCmd := exec.Command(ffmpegPath, "-i", video.CoverPath, "-vf", "scale=48:48:force_original_aspect_ratio=decrease,pad=48:48:(ow-iw)/2:(oh-ih)/2:black", "-y", smallIconPath)
		smallCmd.Run()

		// 生成中图标（80x80）
		mediumIconPath := filepath.Join(iconDir, fmt.Sprintf("icon_%d_medium.png", video.ID))
		mediumCmd := exec.Command(ffmpegPath, "-i", video.CoverPath, "-vf", "scale=80:80:force_original_aspect_ratio=decrease,pad=80:80:(ow-iw)/2:(oh-ih)/2:black", "-y", mediumIconPath)
		mediumCmd.Run()

		// 检查图标是否生成成功
		if _, err := os.Stat(smallIconPath); err == nil {
			if _, err := os.Stat(mediumIconPath); err == nil {
				// 更新数据库中的图标路径
				database.DB.Model(&video).Update("icon_path", iconDir)
				successCount++
			} else {
				failCount++
			}
		} else {
			failCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "图标生成完成",
		"success":   successCount,
		"failed":    failCount,
		"total":     len(videos),
		"icon_path": iconDir,
	})
}

// GenerateSingleIcon 生成单个视频的图标
func GenerateSingleIcon(c *gin.Context) {
	videoID := c.Param("id")

	id, err := strconv.ParseUint(videoID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频ID"})
		return
	}

	var video models.Video
	if err := database.DB.First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 检查是否有封面
	if video.CoverPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "视频没有封面"})
		return
	}

	// 检查封面文件是否存在
	if _, err := os.Stat(video.CoverPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "封面文件不存在"})
		return
	}

	// 确保图标目录存在
	iconDir := "./backend/data/icon"
	if err := os.MkdirAll(iconDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建图标目录失败"})
		return
	}

	// 查找 ffmpeg
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到 ffmpeg"})
		return
	}

	// 生成小图标（48x48）
	smallIconPath := filepath.Join(iconDir, fmt.Sprintf("icon_%d_small.png", video.ID))
	smallCmd := exec.Command(ffmpegPath, "-i", video.CoverPath, "-vf", "scale=48:48:force_original_aspect_ratio=decrease,pad=48:48:(ow-iw)/2:(oh-ih)/2:black", "-y", smallIconPath)
	smallCmd.Run()

	// 生成中图标（80x80）
	mediumIconPath := filepath.Join(iconDir, fmt.Sprintf("icon_%d_medium.png", video.ID))
	mediumCmd := exec.Command(ffmpegPath, "-i", video.CoverPath, "-vf", "scale=80:80:force_original_aspect_ratio=decrease,pad=80:80:(ow-iw)/2:(oh-ih)/2:black", "-y", mediumIconPath)
	mediumCmd.Run()

	// 检查是否生成成功
	if _, err := os.Stat(smallIconPath); err == nil {
		if _, err := os.Stat(mediumIconPath); err == nil {
			// 更新数据库中的图标路径
			database.DB.Model(&video).Update("icon_path", iconDir)

			c.JSON(http.StatusOK, gin.H{
				"message":   "图标生成成功",
				"icon_path": iconDir,
			})
			return
		}
	}

	c.JSON(http.StatusInternalServerError, gin.H{"error": "图标生成失败"})
}

// findFFmpeg 查找 ffmpeg 路径
func findFFmpeg() string {
	paths := []string{
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/opt/homebrew/bin/ffmpeg",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return ""
}
