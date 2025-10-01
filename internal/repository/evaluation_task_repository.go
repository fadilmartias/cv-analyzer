package repository

import (
	"github.com/fadilmartias/cv-analyzer/internal/model"
	"gorm.io/gorm"
)

type EvaluationRepository struct {
	db *gorm.DB
}

func NewEvaluationRepository(db *gorm.DB) *EvaluationRepository {
	return &EvaluationRepository{db}
}

func (r *EvaluationRepository) CreateTask(task *model.EvaluationTask) error {
	return r.db.Create(task).Error
}

func (r *EvaluationRepository) UpdateTask(task *model.EvaluationTask) error {
	return r.db.Save(task).Error
}

func (r *EvaluationRepository) FindTaskByID(id string) (*model.EvaluationTask, error) {
	var task model.EvaluationTask
	err := r.db.First(&task, "id = ?", id).Error
	return &task, err
}
