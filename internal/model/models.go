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
	Category    string    `gorm:"size:100"`
	Rating      float64   `gorm:"default:0"`
	Price       float64   `gorm:"default:0"`
	ImageURL    string    `gorm:"size:500"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

type GuideContent struct {
	ID          uint      `gorm:"primaryKey"`
	SpotID      uint      `gorm:"not null"`
	Title       string    `gorm:"size:255;not null"`
	Content     string    `gorm:"text;not null"`
	Type        string    `gorm:"size:50"`
	AudioURL    string    `gorm:"size:500"`
	Duration    int       `gorm:"default:0"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
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
	ID        uint      `gorm:"primaryKey"`
	Query     string    `gorm:"text;not null"`
	Response  string    `gorm:"text"`
	SpotID    uint      `gorm:"default:0"`
	IsAnswered bool     `gorm:"default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Username  string    `gorm:"size:100;unique;not null"`
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

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ScenicSpot{},
		&GuideContent{},
		&TourRoute{},
		&VisitorQuery{},
		&User{},
		&VisitRecord{},
		&SystemLog{},
	)
}
