package model

import (
	"time"
)

type KnowledgeChunk struct {
	ID                string    `gorm:"primaryKey" json:"id"`
	Content           string    `gorm:"text;not null" json:"content"`
	Source            string    `gorm:"size:255;index:idx_knowledge_chunks_source_updated" json:"source"`
	Title             string    `gorm:"size:255;index:idx_knowledge_chunks_title" json:"title"`
	Metadata          string    `gorm:"text" json:"metadata"`
	KnowledgeCategory string    `gorm:"size:100;index:idx_knowledge_chunks_knowledge_category" json:"knowledge_category"`
	SpotID            uint      `gorm:"index:idx_knowledge_chunks_spot_id" json:"spot_id"`
	SpotCategory      string    `gorm:"size:100;index:idx_knowledge_chunks_spot_category" json:"spot_category"`
	Vector            string    `gorm:"text;not null" json:"-"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime;index:idx_knowledge_chunks_source_updated;index:idx_knowledge_chunks_updated" json:"updated_at"`
}
