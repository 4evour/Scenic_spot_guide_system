package model

import (
	"time"

	"gorm.io/gorm"
)

type ScenicSpot struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:255;not null"`
	Description string    `gorm:"text"`
	Location    string    `gorm:"size:500"`
	Category    string    `gorm:"size:100;index:idx_scenic_spots_category_updated"`
	Rating      float64   `gorm:"default:0"`
	Price       float64   `gorm:"default:0"`
	ImageURL    string    `gorm:"size:500"`
	Latitude    float64   `gorm:"column:latitude;default:0"`
	Longitude   float64   `gorm:"column:longitude;default:0"`
	SortOrder   int       `gorm:"column:sort_order;default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime;index:idx_scenic_spots_category_updated"`
}

type GuideContent struct {
	ID        uint      `gorm:"primaryKey"`
	SpotID    uint      `gorm:"not null"`
	Title     string    `gorm:"size:255;not null"`
	Content   string    `gorm:"text;not null"`
	Type      string    `gorm:"size:50"`
	AudioURL  string    `gorm:"size:500"`
	Duration  int       `gorm:"default:0"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type TourRoute struct {
	ID          uint      `gorm:"primaryKey"`
	Name        string    `gorm:"size:255;not null"`
	Description string    `gorm:"text"`
	Spots       string    `gorm:"text"`
	Duration    int       `gorm:"default:0"`
	Difficulty  string    `gorm:"size:50"`
	Rating      float64   `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

type VisitorQuery struct {
	ID         uint      `gorm:"primaryKey"`
	Query      string    `gorm:"text;not null"`
	Response   string    `gorm:"text"`
	SpotID     uint      `gorm:"default:0"`
	IsAnswered bool      `gorm:"default:false"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"size:100;uniqueIndex;not null"`
	Password  string    `gorm:"size:255;not null"`
	Email     string    `gorm:"size:255"`
	Role      string    `gorm:"size:50;default:'visitor'"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

type VisitRecord struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	SpotID    uint      `gorm:"not null"`
	VisitTime time.Time `gorm:"autoCreateTime"`
	Duration  int       `gorm:"default:0"`
}

type SystemLog struct {
	ID        uint      `gorm:"primaryKey"`
	Level     string    `gorm:"size:50"`
	Message   string    `gorm:"text"`
	Source    string    `gorm:"size:255"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// InteractionLog 交互日志 - 记录每次游客与AI的对话
type InteractionLog struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         uint      `gorm:"default:0"`
	SessionID      string    `gorm:"size:100;index:idx_interaction_logs_session"`
	Query          string    `gorm:"text;not null"`
	Response       string    `gorm:"text"`
	Emotion        string    `gorm:"size:50"`
	ResponseTimeMs int64     `gorm:"default:0"` // 响应时间(毫秒)
	SpotID         uint      `gorm:"default:0"`
	Category       string    `gorm:"size:100"`                                          // 问题分类
	Source         string    `gorm:"size:50;index:idx_interaction_logs_source_created"` // 来源: web/voice/digital_human
	CreatedAt      time.Time `gorm:"autoCreateTime;index:idx_interaction_logs_source_created;index:idx_interaction_logs_created"`
}

// SystemSetting 系统设置 - 键值对存储
type SystemSetting struct {
	ID        uint      `gorm:"primaryKey"`
	Key       string    `gorm:"size:255;unique;not null"`
	Value     string    `gorm:"text"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// DigitalHumanConfig 数字人配置
type DigitalHumanConfig struct {
	ID             uint      `gorm:"primaryKey"`
	Name           string    `gorm:"size:100;default:'小灵'"`
	Appearance     string    `gorm:"size:255;default:'亲和型国风讲解员'"`
	Costume        string    `gorm:"size:255;default:'古典汉服'"`
	Style          string    `gorm:"size:100;default:'古典汉服'"`
	Color          string    `gorm:"size:20;default:'#D4AF37'"`
	CultureTheme   string    `gorm:"size:255;default:'灵山佛教文化与江南山水意境'"`
	VoiceType      string    `gorm:"size:100;default:'温柔女声'"`
	VoiceTone      string    `gorm:"size:100;default:'温暖、端庄、亲切'"`
	Speed          float64   `gorm:"default:0.8"`
	Volume         int       `gorm:"default:80"`
	Greeting       string    `gorm:"size:500;default:'欢迎来到灵山胜境，我是您的数字导览员小灵。'"`
	DefaultEmotion string    `gorm:"size:50;default:'joy'"`
	EmotionLevel   int       `gorm:"default:3"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ScenicSpot{},
		&GuideContent{},
		&TourRoute{},
		&VisitorQuery{},
		&User{},
		&VisitRecord{},
		&SystemLog{},
		&KnowledgeChunk{},
		&InteractionLog{},
		&SystemSetting{},
		&DigitalHumanConfig{},
	)
}
