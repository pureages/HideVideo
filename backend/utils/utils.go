package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"hidevideo/backend/config"
	"strings"
)

// VideoInfo 视频信息
type VideoInfo struct {
	Duration float64 // 时长（秒）
	Width    int     // 宽度
	Height   int     // 高度
	Codec    string  // 编码格式
}

// GetVideoInfo 获取视频信息
func GetVideoInfo(videoPath string) (*VideoInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe error: %v", err)
	}

	info := &VideoInfo{}
	outputStr := string(output)

	// 解析时长 - 匹配 "duration": "xxx" 或 "duration":xxx
	durationRe := regexp.MustCompile(`"duration":\s*"([0-9.]+)"`)
	durationMatch := durationRe.FindStringSubmatch(outputStr)
	if len(durationMatch) > 1 {
		fmt.Sscanf(durationMatch[1], "%f", &info.Duration)
	} else {
		// 尝试不包含引号的格式
		durationRe2 := regexp.MustCompile(`"duration":\s*([0-9.]+)`)
		durationMatch2 := durationRe2.FindStringSubmatch(outputStr)
		if len(durationMatch2) > 1 {
			fmt.Sscanf(durationMatch2[1], "%f", &info.Duration)
		}
	}

	// 解析视频流信息
	// 找到 video 类型流的位置，然后在其后查找 codec, width, height

	// 找到所有包含 codec_name 的行，然后检查前面是否有 video
	lines := strings.Split(outputStr, "\n")
	inVideoStream := false
	for _, line := range lines {
		if strings.Contains(line, `"codec_type": "video"`) {
			inVideoStream = true
			continue
		}
		if inVideoStream && strings.Contains(line, `"codec_type"`) {
			break // 进入下一个流了
		}
		if inVideoStream {
			// 查找 codec_name
			if strings.Contains(line, `"codec_name"`) {
				re := regexp.MustCompile(`"codec_name":\s*"([^"]+)"`)
				m := re.FindStringSubmatch(line)
				if len(m) > 1 {
					info.Codec = m[1]
				}
			}
			// 查找 width
			if strings.Contains(line, `"width"`) && !strings.Contains(line, "coded") {
				re := regexp.MustCompile(`"width":\s*(\d+)`)
				m := re.FindStringSubmatch(line)
				if len(m) > 1 {
					fmt.Sscanf(m[1], "%d", &info.Width)
				}
			}
			// 查找 height
			if strings.Contains(line, `"height"`) && !strings.Contains(line, "coded") {
				re := regexp.MustCompile(`"height":\s*(\d+)`)
				m := re.FindStringSubmatch(line)
				if len(m) > 1 {
					fmt.Sscanf(m[1], "%d", &info.Height)
				}
			}
		}
	}

	return info, nil
}

// GenerateCover 生成视频封面
func GenerateCover(videoPath string, videoID uint, second float64) (string, error) {
	// 确保封面目录存在
	coverDir := config.ServerConfig.StaticPath
	if err := os.MkdirAll(coverDir, 0755); err != nil {
		return "", err
	}

	// 获取视频时长，如果视频不足指定秒数，则使用0
	videoDuration, err := GetVideoInfo(videoPath)
	if err != nil {
		return "", err
	}

	// 如果视频时长小于指定秒数，使用0
	actualSecond := second
	if videoDuration.Duration < second {
		actualSecond = 0
	}

	// 生成封面文件名（使用 jpg 扩展名）
	coverFilename := fmt.Sprintf("cover_%d_%d.jpg", videoID, int(second))
	coverPath := filepath.Join(coverDir, coverFilename)

	// 如果封面已存在，先删除
	if _, err := os.Stat(coverPath); err == nil {
		os.Remove(coverPath)
	}

	// 使用 ffmpeg 截取封面，并缩放到最大 240x140，保持原始宽高比
	cmd := exec.Command("ffmpeg",
		"-y",
		"-ss", fmt.Sprintf("%.2f", actualSecond),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", "scale=240:140:force_original_aspect_ratio=decrease",
		"-q:v", "2",
		coverPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg error: %v", err)
	}

	return coverPath, nil
}

// GetVideoFiles 获取目录下的所有视频文件
func GetVideoFiles(dirPath string) ([]string, error) {
	var videos []string

	videoExtensions := []string{".mp4", ".avi", ".mkv", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, videoExt := range videoExtensions {
			if ext == videoExt {
				videos = append(videos, path)
				break
			}
		}

		return nil
	})

	return videos, err
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsDir 检查是否为目录
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
