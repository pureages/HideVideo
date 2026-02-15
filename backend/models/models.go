package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表
type User struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Password     string         `gorm:"size:255;not null" json:"-"`
	PasswordPlain string        `gorm:"size:255" json:"-"`
	Role        string         `gorm:"size:20;default:'member'" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// VideoLibrary 视频库表
type VideoLibrary struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"size:100;not null;unique" json:"name"`
	Path      string         `gorm:"size:500" json:"path"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Videos    []Video        `gorm:"foreignKey:LibraryID" json:"-"`
}

// Video 视频表
type Video struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	LibraryID  uint           `gorm:"index;not null" json:"library_id"`
	Filename   string         `gorm:"size:255;not null" json:"filename"`
	Filepath   string         `gorm:"size:500;not null" json:"filepath"`
	Duration   float64        `gorm:"default:0" json:"duration"`
	Width      int            `gorm:"default:0" json:"width"`
	Height     int            `gorm:"default:0" json:"height"`
	Codec      string         `gorm:"size:50" json:"codec"`
	CreatedAt  time.Time      `json:"created_at"`
	PlayCount  int            `gorm:"default:0" json:"play_count"`
	Rating     float64        `gorm:"default:0" json:"rating"`
	CoverPath  string         `gorm:"size:500" json:"cover_path"`
	IconPath   string         `gorm:"size:500" json:"icon_path"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
	Library    VideoLibrary   `gorm:"foreignKey:LibraryID" json:"-"`
	Tags       []Tag          `gorm:"many2many:video_tags;" json:"tags"`
	Actors     []Actor       `gorm:"many2many:video_actors;" json:"actors"`
	Comments   []Comment      `gorm:"foreignKey:VideoID" json:"comments"`
}

// Tag 标签表
type Tag struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;size:50;not null" json:"name"`
	SortOrder int            `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Videos    []Video        `gorm:"many2many:video_tags;" json:"-"`
}

// Comment 评论表
type Comment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	VideoID   uint           `gorm:"index;not null" json:"video_id"`
	UserID    uint           `gorm:"index" json:"user_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Video     Video          `gorm:"foreignKey:VideoID" json:"-"`
	User      User           `gorm:"foreignKey:UserID" json:"user"`
}

// VideoTag 视频标签关联表
type VideoTag struct {
	VideoID uint `gorm:"primaryKey" json:"video_id"`
	TagID   uint `gorm:"primaryKey" json:"tag_id"`
}

// Actor 演员表
type Actor struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;size:50;not null" json:"name"`
	SortOrder int            `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Videos    []Video        `gorm:"many2many:video_actors;" json:"-"`
}

// VideoActor 视频演员关联表
type VideoActor struct {
	VideoID uint `gorm:"primaryKey" json:"video_id"`
	ActorID uint `gorm:"primaryKey" json:"actor_id"`
}

// Folder 文件夹表（用于缓存文件夹结构）
type Folder struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	LibraryID uint           `gorm:"index;not null" json:"library_id"`
	Name      string         `gorm:"size:255;not null" json:"name"`
	Path      string         `gorm:"size:500;not null" json:"path"`
	ParentID  *uint          `gorm:"index" json:"parent_id"`
	SortOrder int            `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
