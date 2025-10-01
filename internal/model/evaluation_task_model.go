package model

import (
	"time"

	"github.com/google/uuid"
)

type EvaluationTask struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	CV              string    `gorm:"type:text" json:"cv"`
	Report          string    `gorm:"type:text" json:"report"`
	Status          string    `gorm:"type:varchar(50)" json:"status"` // e.g. "processing", "completed", "failed"
	CvMatchRate     float64   `gorm:"type:float" json:"cv_match_rate"`
	CvFeedback      string    `gorm:"type:text" json:"cv_feedback"`
	ProjectScore    float64   `gorm:"type:float" json:"project_score"`
	ProjectFeedback string    `gorm:"type:text" json:"project_feedback"`
	OverallSummary  string    `gorm:"type:text" json:"overall_summary"`
	Breakdown       string    `gorm:"type:jsonb" json:"breakdown"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
