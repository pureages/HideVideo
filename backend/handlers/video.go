package handlers

import (
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"hidevideo/backend/database"
	"hidevideo/backend/models"
	"hidevideo/backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StreamVideo 视频流式播放
func StreamVideo(c *gin.Context) {
	id := c.Param("id")
	var video models.Video

	if err := database.DB.Preload("Library").First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 设置正确的 Content-Type
	contentType := "video/mp4"
	ext := strings.ToLower(filepath.Ext(video.Filepath))
	switch ext {
	case ".mkv":
		contentType = "video/x-matroska"
	case ".webm":
		contentType = "video/webm"
	case ".avi":
		contentType = "video/x-msvideo"
	case ".mov":
		contentType = "video/quicktime"
	}

	// 如果数据库中的 filepath 指向的是图片等非视频文件，拒绝播放并返回错误
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标文件不是视频"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(video.Filepath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频文件不存在"})
		return
	}

	codec := getVideoCodec(video.Filepath)
	if shouldTranscode(ext, codec) {
		streamTranscodedVideo(c, video.Filepath)
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline")
	c.File(video.Filepath)
}

func shouldTranscode(ext, codec string) bool {
	switch ext {
	case ".mp4":
		return codec != "" && codec != "h264"
	case ".webm", ".ogg":
		return false
	default:
		return true
	}
}

func getVideoCodec(videoPath string) string {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_name",
		"-of", "default=nw=1:nk=1",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func streamTranscodedVideo(c *gin.Context, videoPath string) {
	c.Header("Content-Type", "video/mp4")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "no-store")
	c.Status(http.StatusOK)

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-vcodec", "libx264",
		"-acodec", "aac",
		"-movflags", "frag_keyframe+empty_moov",
		"-f", "mp4",
		"pipe:1",
	)
	cmd.Stdout = c.Writer
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "视频转码失败"})
		return
	}
}

// VideoQueryParams 视频查询参数
type VideoQueryParams struct {
	Page       int      `form:"page"`
	PageSize   int      `form:"page_size"`
	LibraryIDs []uint   `form:"library_ids"`
	TagIDs     []uint   `form:"tag_ids"`
	SortBy     string   `form:"sort_by"`
	Order      string   `form:"order"`
	Keyword    string   `form:"keyword"`
	RandomSeed int64    `form:"random_seed"`
	FolderPath string   `form:"folder_path"`
}

// GetVideos 获取视频列表
func GetVideos(c *gin.Context) {
	params := VideoQueryParams{
		Page:     1,
		PageSize: 20,
		SortBy:   "created_at",
		Order:    "desc",
	}

	// 解析分页参数
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("page_size")); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}

	// 解析排序参数
	if sortBy := c.Query("sort_by"); sortBy != "" {
		params.SortBy = sortBy
	}
	if order := c.Query("order"); order != "" {
		params.Order = order
	}

	// 解析视频库筛选
	if libraryIDs := c.Query("library_ids"); libraryIDs != "" {
		var ids []uint
		for _, id := range strings.Split(libraryIDs, ",") {
			if uid, err := strconv.ParseUint(id, 10, 32); err == nil {
				ids = append(ids, uint(uid))
			}
		}
		params.LibraryIDs = ids
	}

	// 解析标签筛选
	if tagIDs := c.Query("tag_ids"); tagIDs != "" {
		var ids []uint
		for _, id := range strings.Split(tagIDs, ",") {
			if uid, err := strconv.ParseUint(id, 10, 32); err == nil {
				ids = append(ids, uint(uid))
			}
		}
		params.TagIDs = ids
	}

	// 解析搜索关键词
	params.Keyword = c.Query("keyword")

	// 解析文件夹路径
	params.FolderPath = c.Query("folder_path")

	// 构建查询
	query := database.DB.Model(&models.Video{}).Preload("Tags")

	// 视频库筛选
	if len(params.LibraryIDs) > 0 {
		query = query.Where("library_id IN ?", params.LibraryIDs)
	}

	// 文件夹路径筛选（仅当前文件夹，不包含子文件夹）
	if params.FolderPath != "" {
		// 需要匹配直接在当前文件夹下的文件，不包含子文件夹
		// 逻辑：统计 filepath 中 "/" 的数量，应该等于 folder_path 中 "/" 的数量 + 1
		// 例如：/mnt/video_test (1个"/") 下的文件应该有 2个"/"
		// /mnt/video_test/视频2 (2个"/") 下的文件应该有 3个"/"
		searchPath := params.FolderPath + "/%"
		// 计算 folder_path 中的 "/" 数量
		folderSlashCount := strings.Count(params.FolderPath, "/")
		// 文件路径中的 "/" 数量应该等于 folderSlashCount + 1
		query = query.Where("filepath LIKE ? AND LENGTH(filepath) - LENGTH(REPLACE(filepath, '/', '')) = ?", searchPath, folderSlashCount+1)
	}

	// 标签筛选（多标签 AND 筛选）
	if len(params.TagIDs) > 0 {
		// 子查询：获取包含所有指定标签的视频ID
		subQuery := database.DB.Model(&models.VideoTag{}).
			Select("video_id").
			Where("tag_id IN ?", params.TagIDs).
			Group("video_id").
			Having("COUNT(DISTINCT tag_id) = ?", len(params.TagIDs))

		query = query.Where("id IN (?)", subQuery)
	}

	// 关键词搜索
	if params.Keyword != "" {
		// 将关键词按空格分割，支持多关键词搜索（AND 逻辑）
		keywords := strings.Fields(params.Keyword)
		idVal, _ := strconv.ParseUint(params.Keyword, 10, 32)

		// 如果是纯数字，当作ID精确匹配
		if len(keywords) == 1 && idVal > 0 {
			query = query.Where("id = ?", uint(idVal))
		} else {
			// 多关键词或非纯数字搜索
			for _, kw := range keywords {
				keyword := "%" + kw + "%"
				// 搜索文件名或通过 video_tags 表搜索标签名
				tagSubQuery := database.DB.Model(&models.VideoTag{}).
					Select("video_id").
					Joins("JOIN tags ON tags.id = video_tags.tag_id").
					Where("tags.name LIKE ?", keyword)
				query = query.Where("(filename LIKE ? OR id IN (?))", keyword, tagSubQuery)
			}
		}
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 判断是否需要使用智能排序（有关键词搜索且使用默认排序时启用）
	useSmartRank := params.Keyword != "" && params.SortBy == "created_at" && params.Order == "desc"

	// 如果有标签筛选但没有关键词搜索，也需要加载标签用于后续筛选
	loadTagsForSearch := len(params.TagIDs) > 0 && params.Keyword == ""

	var videos []models.Video

	if useSmartRank || loadTagsForSearch {
		// 使用智能排序或需要加载标签：先获取所有匹配的视频（不分页）
		var allVideos []models.Video
		if err := query.Find(&allVideos).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频列表失败"})
			return
		}

		// 构建带标签的视频列表
		videoList := make([]utils.VideoWithTags, len(allVideos))
		for i, v := range allVideos {
			// 获取标签
			var tags []models.Tag
			database.DB.Model(&v).Association("Tags").Find(&tags)
			tagNames := make([]string, len(tags))
			for j, t := range tags {
				tagNames[j] = t.Name
			}
			videoList[i] = utils.VideoWithTags{
				Video:    v,
				TagNames: tagNames,
			}
		}

		if useSmartRank {
			// 执行智能排序
			sortedVideos := utils.SearchRank(utils.SearchRankParams{
				Query:     params.Keyword,
				VideoList: videoList,
			})

			// 分页
			offset := (params.Page - 1) * params.PageSize
			endIdx := offset + params.PageSize
			if endIdx > len(sortedVideos) {
				endIdx = len(sortedVideos)
			}
			if offset < len(sortedVideos) {
				videos = sortedVideos[offset:endIdx]
			} else {
				videos = []models.Video{}
			}
		} else {
			// 只加载标签，不需要智能排序，按默认排序
			orderStr := params.SortBy + " " + params.Order
			if params.SortBy == "random" {
				orderStr = "RANDOM()"
			}

			// 排序
			var sortedList []models.Video
			if orderStr == "RANDOM()" {
				// 随机排序需要在内存中处理
				sortedList = allVideos
				// 使用种子进行确定性随机打乱
				if params.RandomSeed > 0 {
					// 使用固定的种子进行随机打乱，确保相同种子得到相同顺序
					r := rand.New(rand.NewSource(params.RandomSeed))
					r.Shuffle(len(sortedList), func(i, j int) {
						sortedList[i], sortedList[j] = sortedList[j], sortedList[i]
					})
				} else {
					// 无种子时使用完全随机
					for i := len(sortedList) - 1; i > 0; i-- {
						j := int64(i) % int64(i+1)
						sortedList[i], sortedList[j] = sortedList[j], sortedList[i]
					}
				}
			} else {
				// 使用数据库排序
				query.Order(orderStr).Find(&sortedList)
			}

			// 分页
			offset := (params.Page - 1) * params.PageSize
			endIdx := offset + params.PageSize
			if endIdx > len(sortedList) {
				endIdx = len(sortedList)
			}
			if offset < len(sortedList) {
				videos = sortedList[offset:endIdx]
			} else {
				videos = []models.Video{}
			}
		}
	} else {
		// 使用默认排序
		orderStr := params.SortBy + " " + params.Order
		if params.SortBy == "random" {
			orderStr = "RANDOM()"
		}
		query = query.Order(orderStr)

		// 分页
		offset := (params.Page - 1) * params.PageSize

		if err := query.Offset(offset).Limit(params.PageSize).Find(&videos).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频列表失败"})
			return
		}
	}

	// 处理封面路径，转换为相对URL
	for i := range videos {
		if videos[i].CoverPath != "" {
			// 转换为 URL 路径
			videos[i].CoverPath = "/covers/" + getCoverFilename(videos[i].CoverPath)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"list":        videos,
		"total":       total,
		"page":        params.Page,
		"page_size":  params.PageSize,
		"total_pages": (int(total) + params.PageSize - 1) / params.PageSize,
	})
}

// getCoverFilename 从完整路径中获取文件名
func getCoverFilename(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	parts = strings.Split(path, "\\")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

// GetVideo 获取视频详情
func GetVideo(c *gin.Context) {
	id := c.Param("id")
	var video models.Video

	if err := database.DB.Preload("Tags").Preload("Comments").First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 转换封面路径
	if video.CoverPath != "" {
		video.CoverPath = "/covers/" + getCoverFilename(video.CoverPath)
	}

	c.JSON(http.StatusOK, video)
}

// UpdateRating 更新评分
func UpdateRating(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Rating float64 `json:"rating" binding:"required,min=0,max=10"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "评分范围为0-10"})
		return
	}

	var video models.Video
	if err := database.DB.First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	database.DB.Model(&video).Update("rating", req.Rating)

	c.JSON(http.StatusOK, gin.H{"message": "评分更新成功"})
}

// UpdateVideoFilename 更新视频文件名
func UpdateVideoFilename(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Filename string `json:"filename" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入文件名"})
		return
	}

	if req.Filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名不能为空"})
		return
	}

	var video models.Video
	if err := database.DB.Preload("Library").First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 获取旧的目录路径
	oldDir := ""
	if video.Filepath != "" {
		lastSlash := strings.LastIndex(video.Filepath, "/")
		if lastSlash > 0 {
			oldDir = video.Filepath[:lastSlash+1]
		}
	}

	// 保留原来的扩展名
	oldExt := ""
	if video.Filename != "" {
		dotIndex := strings.LastIndex(video.Filename, ".")
		if dotIndex > 0 {
			oldExt = video.Filename[dotIndex:]
		}
	}

	// 始终保留原来的扩展名
	newFilename := req.Filename
	if oldExt != "" {
		newFilename = newFilename + oldExt
	}

	// 更新filepath
	newFilepath := oldDir + newFilename

	// 如果文件路径没有变化，只更新数据库
	if newFilepath == video.Filepath {
		database.DB.Model(&video).Update("filename", newFilename)
		c.JSON(http.StatusOK, gin.H{"message": "文件名更新成功", "video": video})
		return
	}

	// 检查原文件是否存在
	if _, err := os.Stat(video.Filepath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "原视频文件不存在，无法重命名"})
		return
	}

	// 尝试重命名磁盘上的文件
	if err := os.Rename(video.Filepath, newFilepath); err != nil {
		// 检查是否为权限错误
		if os.IsPermission(err) {
			c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，无法修改！"})
			return
		}
		// 其他错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件重命名失败: " + err.Error()})
		return
	}

	// 更新数据库
	database.DB.Model(&video).Updates(map[string]interface{}{
		"filename": newFilename,
		"filepath": newFilepath,
	})

	c.JSON(http.StatusOK, gin.H{"message": "文件名更新成功", "video": video})
}

// IncrementPlayCount 增加播放次数
func IncrementPlayCount(c *gin.Context) {
	id := c.Param("id")
	var video models.Video

	if err := database.DB.First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	database.DB.Model(&video).UpdateColumn("play_count", gorm.Expr("play_count + ?", 1))

	c.JSON(http.StatusOK, gin.H{"message": "播放次数已更新"})
}

// DeleteVideo 删除视频
func DeleteVideo(c *gin.Context) {
	id := c.Param("id")
	var video models.Video

	if err := database.DB.First(&video, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "视频不存在"})
		return
	}

	// 删除磁盘上的视频文件
	if video.Filepath != "" {
		if _, err := os.Stat(video.Filepath); err == nil {
			if err := os.Remove(video.Filepath); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "删除视频文件失败: " + err.Error()})
				return
			}
		}
	}

	// 删除封面文件
	if video.CoverPath != "" {
		if _, err := os.Stat(video.CoverPath); err == nil {
			os.Remove(video.CoverPath)
		}
	}

	// 删除视频标签关联
	database.DB.Where("video_id = ?", video.ID).Delete(&models.VideoTag{})

	// 删除评论
	database.DB.Where("video_id = ?", video.ID).Delete(&models.Comment{})

	// 删除视频记录
	if err := database.DB.Delete(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除视频记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "视频删除成功"})
}

// OldFolderInfo 旧版文件夹信息（用于GetFolderTree兼容）
type OldFolderInfo struct {
	Name      string            `json:"name"`
	Path      string            `json:"path"`
	LibraryID uint              `json:"library_id"`
	Children  []OldFolderInfo   `json:"children,omitempty"`
}

// GetFolderTree 获取文件夹树结构
func GetFolderTree(c *gin.Context) {
	// 获取所有视频库
	var libraries []models.VideoLibrary
	if err := database.DB.Find(&libraries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取视频库失败"})
		return
	}

	// 构建文件夹树
	var folderTree []OldFolderInfo

	for _, lib := range libraries {
		if lib.Path == "" {
			continue
		}

		// 获取该库下所有视频的目录
		var folders []string
		database.DB.Model(&models.Video{}).
			Where("library_id = ?", lib.ID).
			Distinct("SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1)").
			Pluck("SUBSTR(filepath, 1, LENGTH(filepath) - LENGTH(filename) - 1)", &folders)

		// 构建该库的文件夹结构
		libraryFolder := OldFolderInfo{
			Name:      lib.Name,
			Path:      lib.Path,
			LibraryID: lib.ID,
			Children:  buildFolderChildrenFromSlice(folders, lib.Path),
		}

		// 如果有子文件夹或该库本身有视频，则添加
		if len(libraryFolder.Children) > 0 || len(folders) > 0 {
			folderTree = append(folderTree, libraryFolder)
		}
	}

	c.JSON(http.StatusOK, folderTree)
}

// buildFolderChildrenFromSlice 从字符串切片构建子文件夹
func buildFolderChildrenFromSlice(folders []string, basePath string) []OldFolderInfo {
	// 使用map来组织文件夹结构
	folderMap := make(map[string]*OldFolderInfo)

	// 首先创建所有文件夹节点
	for _, f := range folders {
		if f == "" || f == basePath {
			continue
		}

		// 计算相对路径
		relPath := f
		if strings.HasPrefix(relPath, basePath) {
			relPath = strings.TrimPrefix(relPath, basePath)
			relPath = strings.TrimPrefix(relPath, "/")
		}

		parts := strings.Split(relPath, "/")
		currentPath := basePath

		for i, part := range parts {
			if part == "" {
				continue
			}

			if i > 0 {
				currentPath += "/" + part
			} else {
				currentPath = basePath + "/" + part
			}

			if _, exists := folderMap[currentPath]; !exists {
				folder := OldFolderInfo{
					Name:     part,
					Path:     currentPath,
					Children: []OldFolderInfo{},
				}
				folderMap[currentPath] = &folder
			}
		}
	}

	// 然后构建父子关系
	for path, folder := range folderMap {
		// 计算父路径
		lastSlash := strings.LastIndex(path, "/")
		parentPath := path[:lastSlash]

		if parentPath != basePath {
			// 找到父文件夹并添加子文件夹
			if parent, exists := folderMap[parentPath]; exists {
				parent.Children = append(parent.Children, *folder)
			}
		}
	}

	// 最后提取根文件夹
	var rootFolders []OldFolderInfo
	for _, folder := range folderMap {
		lastSlash := strings.LastIndex(folder.Path, "/")
		parentPath := folder.Path[:lastSlash]
		if parentPath == basePath {
			rootFolders = append(rootFolders, *folder)
		}
	}

	// 按名称排序
	sortOldFolders(rootFolders)
	return rootFolders
}

// sortOldFolders 按名称排序文件夹
func sortOldFolders(folders []OldFolderInfo) {
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].Name < folders[j].Name
	})
	for i := range folders {
		if len(folders[i].Children) > 0 {
			sortOldFolders(folders[i].Children)
		}
	}
}

// GetVideoByPath 通过文件路径获取视频（用于文件管理器播放未入库的视频）
func GetVideoByPath(c *gin.Context) {
	filepath := c.Query("filepath")
	libraryId := c.Query("library_id")

	if filepath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少文件路径"})
		return
	}

	// 先尝试查找已存在的视频
	var video models.Video
	err := database.DB.Where("filepath = ?", filepath).First(&video).Error
	if err == nil {
		// 找到已存在的视频
		c.JSON(http.StatusOK, gin.H{
			"video":    video,
			"is_new":   false,
		})
		return
	}

	// 未找到，创建新的视频记录
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少视频库ID"})
		return
	}

	// 解析 library_id
	libID, err := strconv.ParseUint(libraryId, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的视频库ID"})
		return
	}

	// 获取视频信息
	videoInfo, err := utils.GetVideoInfo(filepath)
	if err != nil {
		// 如果无法获取视频信息，使用默认值创建
		video = models.Video{
			LibraryID: uint(libID),
			Filename:  path.Base(filepath),
			Filepath:  filepath,
		}
	} else {
		video = models.Video{
			LibraryID: uint(libID),
			Filename:  path.Base(filepath),
			Filepath:  filepath,
			Duration:  videoInfo.Duration,
			Width:     videoInfo.Width,
			Height:    videoInfo.Height,
			Codec:     videoInfo.Codec,
		}
	}

	// 保存到数据库
	if err := database.DB.Create(&video).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建视频记录失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video":  video,
		"is_new": true,
	})
}
