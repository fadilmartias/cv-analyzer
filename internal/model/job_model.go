package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Job struct {
	ID        uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	Title     string          `json:"title"`
	Content   string          `gorm:"type:text" json:"content"`
	Embedding pgvector.Vector `gorm:"type:vector(3072)" json:"embedding"` // pakai pgvector
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (j *Job) TableName() string {
	return "jobs"
}
