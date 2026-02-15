package database

import (
	"hidevideo/backend/config"
	"hidevideo/backend/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

// Init 初始化数据库
func Init() error {
	var err error
	DB, err = gorm.Open(sqlite.Open(config.DatabaseConfig.Path), &gorm.Config{})
	if err != nil {
		return err
	}

	// 自动迁移
	if err := DB.AutoMigrate(
		&models.User{},
		&models.VideoLibrary{},
		&models.Video{},
		&models.Tag{},
		&models.Comment{},
		&models.VideoTag{},
		&models.Actor{},
		&models.VideoActor{},
		&models.Folder{},
	); err != nil {
		return err
	}

	// 创建默认用户
	createDefaultUser()

	return nil
}

// createDefaultUser 创建默认用户
func createDefaultUser() {
	var count int64
	DB.Model(&models.User{}).Count(&count)
	if count == 0 {
		// 默认用户名: admin, 密码: admin123
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		defaultUser := models.User{
			Username:      "admin",
			Password:      string(hashedPassword),
			PasswordPlain: "admin123",
			Role:         "admin",
		}
		DB.Create(&defaultUser)
	}
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
