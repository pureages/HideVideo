package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"hidevideo/backend/database"
	"hidevideo/backend/models"

	"github.com/gin-gonic/gin"
)

// FileInfo 表示文件信息
type FileInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
	Extension string    `json:"extension,omitempty"`
}

// ListLibraryFiles 列出视频库目录下的文件
func ListLibraryFiles(c *gin.Context) {
	libraryID := c.Param("id")
	path := c.Query("path")

	// 获取视频库信息
	var library struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	}

	// 从数据库获取视频库路径
	err := database.DB.Model(&models.VideoLibrary{}).Where("id = ?", libraryID).First(&library).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	// 确定要列出的目录
	dirPath := library.Path
	if path != "" {
		// 验证路径在视频库目录内（安全检查）
		absLibraryPath, _ := filepath.Abs(library.Path)
		absRequestPath, _ := filepath.Abs(path)

		// 检查请求的路径是否在视频库路径内
		if len(absRequestPath) < len(absLibraryPath) || absRequestPath[:len(absLibraryPath)] != absLibraryPath {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的路径"})
			return
		}
		dirPath = path
	}

	// 检查目录是否存在
	info, err := os.Stat(dirPath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "目录不存在"})
		return
	}

	if !info.IsDir() {
		// 如果是文件，返回文件所在的目录
		dirPath = filepath.Dir(dirPath)
	}

	// 读取目录
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取目录失败: " + err.Error()})
		return
	}

	var files []FileInfo
	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		f := FileInfo{
			Name:    entry.Name(),
			Path:    fullPath,
			IsDir:   entry.IsDir(),
			Size:    fileInfo.Size(),
			ModTime: fileInfo.ModTime(),
		}

		if !entry.IsDir() {
			f.Extension = filepath.Ext(entry.Name())
		}

		files = append(files, f)
	}

	// 返回相对于视频库的路径
	relativePath := dirPath
	if len(dirPath) > len(library.Path) {
		relativePath = dirPath[len(library.Path):]
		if relativePath != "" && relativePath[0] == filepath.Separator {
			relativePath = relativePath[1:]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"files":        files,
		"path":         dirPath,
		"relativePath": relativePath,
		"library":      library,
	})
}

// GetLibraryPath 获取视频库的根路径
func GetLibraryPath(c *gin.Context) {
	libraryID := c.Param("id")

	var library models.VideoLibrary
	err := database.DB.First(&library, libraryID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频库不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":   library.ID,
		"name": library.Name,
		"path": library.Path,
	})
}
