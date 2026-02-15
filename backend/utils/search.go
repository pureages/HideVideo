package utils

import (
	"math"
	"hidevideo/backend/models"
	"strings"
)

// VideoWithTags 带标签的视频结构，用于排序计算
type VideoWithTags struct {
	Video        models.Video
	TagNames     []string
}

// SearchRankParams 搜索排序参数
type SearchRankParams struct {
	Query       string
	VideoList   []VideoWithTags
}

// KeywordInfo 关键词信息
type KeywordInfo struct {
	Word       string
	Weight     float64
	IsCommon   bool  // 是否为大众词
	IsRare     bool  // 是否为稀有词
}

// CalculateKeywordWeights 计算关键词权重
func CalculateKeywordWeights(query string, videoList []VideoWithTags) []KeywordInfo {
	// 分词：按空格分割
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil
	}

	// 统计每个词在视频中的出现频率
	wordCount := make(map[string]int)
	totalVideos := len(videoList)

	for _, video := range videoList {
		// 检查标题
		titleLower := strings.ToLower(video.Video.Filename)

		// 检查标签
		var tagsText string
		for _, tag := range video.TagNames {
			tagsText += " " + strings.ToLower(tag)
		}

		seen := make(map[string]bool)
		for _, word := range words {
			wordLower := strings.ToLower(word)
			if strings.Contains(titleLower, wordLower) || strings.Contains(tagsText, wordLower) {
				if !seen[wordLower] {
					wordCount[wordLower]++
					seen[wordLower] = true
				}
			}
		}
	}

	// 计算权重
	keywords := make([]KeywordInfo, len(words))
	for i, word := range words {
		wordLower := strings.ToLower(word)
		frequency := float64(wordCount[wordLower]) / float64(totalVideos)

		// 基础权重
		weight := 1.0
		if i == 0 {
			weight = 1.2 // 第一个词为核心词
		}

		// 稀有度检测
		isCommon := false
		isRare := false
		if totalVideos > 0 {
			if frequency > 0.2 {
				weight *= 0.8 // 大众词
				isCommon = true
			} else if frequency < 0.05 && frequency > 0 {
				weight *= 1.5 // 稀有词
				isRare = true
			}
		}

		keywords[i] = KeywordInfo{
			Word:     wordLower,
			Weight:   weight,
			IsCommon: isCommon,
			IsRare:   isRare,
		}
	}

	return keywords
}

// CalculateTitleScore 计算标题得分
func CalculateTitleScore(title string, keyword KeywordInfo) (float64, bool) {
	titleLower := strings.ToLower(title)

	// 检查是否命中
	if !strings.Contains(titleLower, keyword.Word) {
		return 0, false
	}

	// 基础得分
	score := 10.0 * keyword.Weight

	// 位置衰减：计算关键词在标题中的位置
	idx := strings.Index(titleLower, keyword.Word)
	position := float64(idx)

	// 位置衰减：每往后10个字符衰减5%
	decayFactor := 1.0 - 0.05*math.Floor(position/10.0)
	if decayFactor < 0.5 {
		decayFactor = 0.5
	}
	score *= decayFactor

	// 检查连续匹配（简化版：检查关键词前后是否有相邻的词）
	// 这里简化处理，如果关键词在标题开头或靠近开头，给与一定加成
	if position < 10 {
		score *= 1.1
	}

	return score, true
}

// CalculateTagScore 计算标签得分
func CalculateTagScore(tagNames []string, keyword KeywordInfo) float64 {
	for _, tag := range tagNames {
		tagLower := strings.ToLower(tag)
		if strings.Contains(tagLower, keyword.Word) {
			return 5.0 * keyword.Weight
		}
	}
	return 0
}

// SearchRank 搜索排序主函数
func SearchRank(params SearchRankParams) []models.Video {
	if params.Query == "" || len(params.VideoList) == 0 {
		// 无搜索词时返回原列表
		result := make([]models.Video, len(params.VideoList))
		for i, v := range params.VideoList {
			result[i] = v.Video
		}
		return result
	}

	// 计算关键词权重
	keywords := CalculateKeywordWeights(params.Query, params.VideoList)
	if keywords == nil {
		result := make([]models.Video, len(params.VideoList))
		for i, v := range params.VideoList {
			result[i] = v.Video
		}
		return result
	}

	// 计算每个视频的得分
	type scoredVideo struct {
		video       models.Video
		score       float64
		createdAt   int64
	}

	scoredVideos := make([]scoredVideo, len(params.VideoList))

	for i, video := range params.VideoList {
		baseScore := 0.0
		titleMatched := make(map[string]bool)  // 记录哪些关键词在标题中命中
		tagMatched := make(map[string]bool)    // 记录哪些关键词在标签中命中

		for _, kw := range keywords {
			// 标题匹配
			titleScore, titleHit := CalculateTitleScore(video.Video.Filename, kw)
			if titleHit {
				baseScore += titleScore
				titleMatched[kw.Word] = true
			}

			// 标签匹配
			tagScore := CalculateTagScore(video.TagNames, kw)
			if tagScore > 0 {
				baseScore += tagScore
				tagMatched[kw.Word] = true
			}

			// 双重命中补偿：同一关键词同时出现在标题和标签中
			if titleHit && tagScore > 0 {
				baseScore += 3.0 * kw.Weight
			}
		}

		// 热度融合：log10(play_count + 1)
		playCountBonus := math.Log10(float64(video.Video.PlayCount) + 1)

		// 最终得分
		finalScore := baseScore + playCountBonus

		scoredVideos[i] = scoredVideo{
			video:     video.Video,
			score:     finalScore,
			createdAt: video.Video.CreatedAt.Unix(),
		}
	}

	// 排序：第一优先级按得分降序，第二优先级按发布时间降序
	for i := 0; i < len(scoredVideos)-1; i++ {
		for j := i + 1; j < len(scoredVideos); j++ {
			if scoredVideos[j].score > scoredVideos[i].score ||
				(scoredVideos[j].score == scoredVideos[i].score && scoredVideos[j].createdAt > scoredVideos[i].createdAt) {
				scoredVideos[i], scoredVideos[j] = scoredVideos[j], scoredVideos[i]
			}
		}
	}

	// 提取排序后的视频列表
	result := make([]models.Video, len(scoredVideos))
	for i, sv := range scoredVideos {
		result[i] = sv.video
	}

	return result
}
