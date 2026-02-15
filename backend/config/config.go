package config

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ServerConfig 服务器配置
	ServerConfig = struct {
		Port       string
		StaticPath string
		UploadPath string
	}{
		Port:       "49377",
		StaticPath: "./data/covers",
		UploadPath: "./data",
	}

	// DatabaseConfig 数据库配置
	DatabaseConfig = struct {
		Path string
	}{
		Path: "./data/hidevideo.db",
	}

	// SessionConfig Session配置
	SessionConfig = struct {
		Secret string
		MaxAge  int
	}{
		Secret: "hidevideo_secret_key_2024",
		MaxAge: 86400 * 7, // 7天
	}

	// LoginProtectionConfig 登录保护配置
	LoginProtectionConfig = struct {
		Enabled       bool
		MaxAttempts   int
		LockoutTime   time.Duration
		mu            sync.RWMutex
	}{
		Enabled:     false,
		MaxAttempts: 3,
		LockoutTime: 10 * time.Minute,
	}
)

// GetLoginProtectionEnabled 获取登录保护是否开启
func GetLoginProtectionEnabled() bool {
	LoginProtectionConfig.mu.RLock()
	defer LoginProtectionConfig.mu.RUnlock()
	return LoginProtectionConfig.Enabled
}

// SetLoginProtectionEnabled 设置登录保护是否开启
func SetLoginProtectionEnabled(enabled bool) {
	LoginProtectionConfig.mu.Lock()
	defer LoginProtectionConfig.mu.Unlock()
	LoginProtectionConfig.Enabled = enabled
}

// GetMaxAttempts 获取最大登录尝试次数
func GetMaxAttempts() int {
	LoginProtectionConfig.mu.RLock()
	defer LoginProtectionConfig.mu.RUnlock()
	return LoginProtectionConfig.MaxAttempts
}

// GetLockoutTime 获取锁定时间
func GetLockoutTime() time.Duration {
	LoginProtectionConfig.mu.RLock()
	defer LoginProtectionConfig.mu.RUnlock()
	return LoginProtectionConfig.LockoutTime
}

func init() {
	// 确保数据目录存在
	if err := os.MkdirAll(ServerConfig.StaticPath, 0755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Dir(DatabaseConfig.Path), 0755); err != nil {
		panic(err)
	}
}
