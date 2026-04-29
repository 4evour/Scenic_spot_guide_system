package model

import (
	"time"
)

type KnowledgeChunk struct {
	ID       string    `gorm:"primaryKey"`
	Content  string    `gorm:"text;not null"`
	Source   string    `gorm:"size:255"`
	Title    string    `gorm:"size:255"`
	Metadata string    `gorm:"text"`
	Vector   string    `gorm:"text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}