package model

import (
	"time"
)

type KnowledgeChunk struct {
	ID        string    `gorm:"primaryKey"`
	Content   string    `gorm:"text;not null"`
	Source    string    `gorm:"size:255;index:idx_knowledge_chunks_source_updated"`
	Title     string    `gorm:"size:255;index:idx_knowledge_chunks_title"`
	Metadata  string    `gorm:"text"`
	Vector    string    `gorm:"text;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime;index:idx_knowledge_chunks_source_updated;index:idx_knowledge_chunks_updated"`
}
