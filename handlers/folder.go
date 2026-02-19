package handlers

import (
	"hidevideo/backend/database"
	"hidevideo/backend/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// FolderInfo 前端使用的文件夹信息
type FolderInfo struct {
	ID        uint         `json:"id"`
	LibraryID uint         `json:"library_id"`
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	ParentID  *uint        `json:"parent_id"`
	Children  []FolderInfo `json:"children"`
}

// GetFolders 获取所有文件夹（带层级结构）
func GetFolders(c *gin.Context) {
	var folders []models.Folder
	if err := database.DB.Where("parent_id IS NULL").
		Order("sort_order, name").
		Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件夹失败"})
		return
	}

	// 构建层级结构
	var result []FolderInfo
	for _, folder := range folders {
		result = append(result, buildFolderTree(folder))
	}

	c.JSON(http.StatusOK, result)
}

// buildFolderTree 递归构建文件夹树
func buildFolderTree(folder models.Folder) FolderInfo {
	var children []models.Folder
	database.DB.Where("parent_id = ?", folder.ID).
		Order("sort_order, name").
		Find(&children)

	var childInfos []FolderInfo
	for _, child := range children {
		childInfos = append(childInfos, buildFolderTree(child))
	}

	return FolderInfo{
		ID:        folder.ID,
		LibraryID: folder.LibraryID,
		Name:      folder.Name,
		Path:      folder.Path,
		ParentID:  folder.ParentID,
		Children:  childInfos,
	}
}

// GetFoldersByLibrary 按库获取文件夹
func GetFoldersByLibrary(c *gin.Context) {
	libraryID := c.Query("library_id")

	var folders []models.Folder
	query := database.DB.Order("sort_order, name")

	if libraryID != "" {
		query = query.Where("library_id = ?", libraryID)
	}

	if err := query.Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件夹失败"})
		return
	}

	// 构建层级结构
	var rootFolders []models.Folder
	for _, f := range folders {
		if f.ParentID == nil {
			rootFolders = append(rootFolders, f)
		}
	}

	var result []FolderInfo
	for _, folder := range rootFolders {
		result = append(result, buildFolderTree(folder))
	}

	c.JSON(http.StatusOK, result)
}

// TraverseFolders 遍历并保存文件夹结构
func TraverseFolders(c *gin.Context) {
	// 获取所有视频库
	var libraries []models.VideoLibrary
	if err := database.DB.Find(&libraries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频库失败"})
		return
	}

	totalFolders := 0

	for _, lib := range libraries {
		if lib.Path == "" {
			continue
		}

		// 获取该库下所有视频的目录（去重）
		type FolderPath struct {
			Path string
		}
		var folders []FolderPath
		database.DB.Model(&models.Video{}).
			Where("library_id = ?", lib.ID).
			Distinct("SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1)").
			Pluck("SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1)", &folders)

		// 清空该库现有的文件夹记录
		database.DB.Where("library_id = ?", lib.ID).Delete(&models.Folder{})

		// 创建根文件夹（使用真实路径的文件夹名）
		rootFolderName := extractFolderName(lib.Path)
		rootFolder := models.Folder{
			LibraryID: lib.ID,
			Name:      rootFolderName,
			Path:      lib.Path,
			ParentID:  nil,
			SortOrder: 0,
		}
		if err := database.DB.Create(&rootFolder).Error; err != nil {
			continue
		}

		// 构建文件夹映射
		folderMap := make(map[string]*models.Folder)
		folderMap[lib.Path] = &rootFolder

		// 处理每个文件夹路径
		for _, fp := range folders {
			if fp.Path == "" || fp.Path == lib.Path {
				continue
			}

			// 计算相对路径
			relPath := fp.Path
			if strings.HasPrefix(relPath, lib.Path) {
				relPath = strings.TrimPrefix(relPath, lib.Path)
				relPath = strings.TrimPrefix(relPath, "/")
			}

			// 逐层创建文件夹
			parts := strings.Split(relPath, "/")
			currentPath := lib.Path
			var currentParentID *uint
			currentParentID = &rootFolder.ID

			for i, part := range parts {
				if part == "" {
					continue
				}

				if i > 0 {
					currentPath = currentPath + "/" + part
				} else {
					currentPath = lib.Path + "/" + part
				}

				// 检查是否已存在
				if existing, ok := folderMap[currentPath]; ok {
					currentParentID = &existing.ID
					continue
				}

				// 创建新文件夹
				newFolder := models.Folder{
					LibraryID: lib.ID,
					Name:      part,
					Path:      currentPath,
					ParentID:  currentParentID,
					SortOrder: i,
				}

				if err := database.DB.Create(&newFolder).Error; err != nil {
					continue
				}

				folderMap[currentPath] = &newFolder
				currentParentID = &newFolder.ID
				totalFolders++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "文件夹遍历完成",
		"total_folders": totalFolders + len(libraries),
	})
}

// GetVideosByFolderID 根据文件夹ID获取视频（仅当前层级，不含子文件夹）
func GetVideosByFolderID(c *gin.Context) {
	folderID := c.Param("id")

	var folder models.Folder
	if err := database.DB.First(&folder, folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件夹不存在"})
		return
	}

	// 获取该文件夹下的直接视频（不包含子文件夹）
	var videos []models.Video
	if err := database.DB.
		Where("library_id = ? AND SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1) = ?",
			folder.LibraryID, folder.Path).
		Preload("Tags").
		Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频失败"})
		return
	}

	c.JSON(http.StatusOK, videos)
}

// GetFolderByID 获取单个文件夹信息（包含子文件夹）
func GetFolderByID(c *gin.Context) {
	folderID := c.Param("id")

	var folder models.Folder
	if err := database.DB.First(&folder, folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件夹不存在"})
		return
	}

	// 递归获取子文件夹
	children := getFolderChildren(folder.ID)

	c.JSON(http.StatusOK, FolderInfo{
		ID:        folder.ID,
		LibraryID: folder.LibraryID,
		Name:      folder.Name,
		Path:      folder.Path,
		ParentID:  folder.ParentID,
		Children:  children,
	})
}

// getFolderChildren 递归获取子文件夹
func getFolderChildren(parentID uint) []FolderInfo {
	var folders []models.Folder
	database.DB.Where("parent_id = ?", parentID).
		Order("sort_order, name").
		Find(&folders)

	var result []FolderInfo
	for _, f := range folders {
		children := getFolderChildren(f.ID)
		result = append(result, FolderInfo{
			ID:        f.ID,
			LibraryID: f.LibraryID,
			Name:      f.Name,
			Path:      f.Path,
			ParentID:  f.ParentID,
			Children:  children,
		})
	}
	return result
}

// GetAllFoldersFlat 获取所有文件夹（扁平结构，带父子关系）
func GetAllFoldersFlat(c *gin.Context) {
	var folders []models.Folder
	if err := database.DB.Order("library_id, sort_order, name").Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件夹失败"})
		return
	}

	c.JSON(http.StatusOK, folders)
}

// GetVideosInFolder 获取文件夹中的视频（支持分页）
func GetVideosInFolder(c *gin.Context) {
	folderID := c.Param("id")
	page := c.DefaultQuery("page", "1")
	pageSize := c.DefaultQuery("page_size", "24")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	order := c.DefaultQuery("order", "desc")
	keyword := c.Query("keyword")

	var folder models.Folder
	if err := database.DB.First(&folder, folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件夹不存在"})
		return
	}

	// 构建查询
	query := database.DB.Model(&models.Video{}).
		Where("library_id = ? AND SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1) = ?",
			folder.LibraryID, folder.Path)

	// 关键词搜索
	if keyword != "" {
		keywords := strings.Fields(keyword)
		for _, kw := range keywords {
			keywordPattern := "%" + kw + "%"
			query = query.Where("filename LIKE ?", keywordPattern)
		}
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 排序
	orderClause := sortBy + " " + order
	if sortBy == "random" {
		orderClause = "RANDOM()"
	}

	// 分页
	var videos []models.Video
	offset := (parseInt(page) - 1) * parseInt(pageSize)
	if err := query.
		Preload("Tags").
		Order(orderClause).
		Offset(offset).
		Limit(parseInt(pageSize)).
		Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频失败"})
		return
	}

	totalPages := (int(total) + parseInt(pageSize) - 1) / parseInt(pageSize)

	c.JSON(http.StatusOK, gin.H{
		"list":        videos,
		"total":       total,
		"page":        parseInt(page),
		"total_pages": totalPages,
	})
}

func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// GetFolderVideoCount 获取文件夹中的视频数量
func GetFolderVideoCount(c *gin.Context) {
	folderID := c.Param("id")

	var folder models.Folder
	if err := database.DB.First(&folder, folderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件夹不存在"})
		return
	}

	var count int64
	database.DB.Model(&models.Video{}).
		Where("library_id = ? AND SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1) = ?",
			folder.LibraryID, folder.Path).
		Count(&count)

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// extractFolderName 从路径中提取文件夹名
func extractFolderName(path string) string {
	if path == "" {
		return ""
	}
	// 去除末尾的斜杠
	path = strings.TrimRight(path, "/")
	// 获取最后一个斜杠后的部分
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}
