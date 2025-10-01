package repository

import (
	"github.com/fadilmartias/cv-analyzer/internal/model"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type JobRepository struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db}
}

func (r *JobRepository) SearchJobs(embedding pgvector.Vector, topK int) ([]model.Job, error) {
	var jobs []model.Job

	// query pgvector <-> operator (Euclidean distance / cosine)
	err := r.db.Raw(`
        SELECT *, embedding <-> ? AS distance
        FROM jobs
        ORDER BY embedding <-> ?
        LIMIT ?
    `, embedding, embedding, topK).Scan(&jobs).Error

	return jobs, err
}

func (r *JobRepository) CreateJob(job *model.Job) error {
	return r.db.Create(job).Error
}

func (r *JobRepository) UpdateJob(job *model.Job) error {
	return r.db.Save(job).Error
}

func (r *JobRepository) FindJobByID(id string) (*model.Job, error) {
	var j model.Job
	err := r.db.First(&j, "id = ?", id).Error
	return &j, err
}

func (r *JobRepository) GetJobs() ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Find(&jobs).Error
	return jobs, err
}
