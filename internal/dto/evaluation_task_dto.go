package dto

import (
	"time"

	"github.com/google/uuid"
)

type EvaluationTaskDTO struct {
	ID              uuid.UUID `json:"id"`
	Status          string    `json:"status"` // e.g. "processing", "completed", "failed"
	CvMatchRate     float64   `json:"cv_match_rate"`
	CvFeedback      string    `json:"cv_feedback"`
	ProjectScore    float64   `json:"project_score"`
	ProjectFeedback string    `json:"project_feedback"`
	OverallSummary  string    `json:"overall_summary"`
	Breakdown       string    `json:"breakdown"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
