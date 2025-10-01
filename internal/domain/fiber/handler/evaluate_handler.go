package handler

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/fadilmartias/cv-analyzer/internal/dto"
	"github.com/fadilmartias/cv-analyzer/internal/middleware"
	"github.com/fadilmartias/cv-analyzer/internal/model"
	"github.com/fadilmartias/cv-analyzer/internal/usecase"
	"github.com/fadilmartias/cv-analyzer/internal/util"
	"github.com/gofiber/fiber/v2"
)

type EvaluateHandler struct {
	uc *usecase.EvaluationUsecase
}

func NewEvaluateHandler(uc *usecase.EvaluationUsecase) *EvaluateHandler {
	return &EvaluateHandler{uc: uc}
}

func (h *EvaluateHandler) RegisterRoutes(app *fiber.App) {
	app.Post("/evaluate", middleware.RateLimiter(1, 4*time.Second), h.Evaluate)
	app.Get("/result/:id", h.Result)
	app.Get("/test", h.Test)
	app.Get("/create-job-embedding", h.CreateJobEmbedding)
}

func (h *EvaluateHandler) Evaluate(c *fiber.Ctx) error {
	cvContent, err := h.processFile(c, "cv", "./uploads/cv/")
	if err != nil {
		return err
	}

	reportContent, err := h.processFile(c, "project_report", "./uploads/project_report/")
	if err != nil {
		return err
	}

	log.Println("CV Content:", cvContent)
	log.Println("Report Content:", reportContent)

	task := model.EvaluationTask{
		CV:     cvContent,
		Report: reportContent,
	}

	id, err := h.uc.Submit(task)
	if err != nil {
		return util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: "failed to submit evaluation",
		}, err)
	}

	return util.SuccessResponse(c, util.SuccessResponseFormat{
		Message: "Success submit evaluation",
		Data:    fiber.Map{"id": id, "status": "processing"},
	})
}

func (h *EvaluateHandler) CreateJobEmbedding(c *fiber.Ctx) error {
	if err := h.uc.CreateJobEmbedding(); err != nil {
		return util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: "failed to create job embedding",
		}, err)
	}
	return util.SuccessResponse(c, util.SuccessResponseFormat{
		Message: "Success create job embedding",
	})
}

func (h *EvaluateHandler) processFile(c *fiber.Ctx, fieldName, uploadDir string) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: fmt.Sprintf("%s file is required", fieldName),
		}, err)
	}

	fileSize := file.Size
	if fileSize > 5*1024*1024 {
		return "", util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: fmt.Sprintf("%s file size is too large (max 5MB)", fieldName),
		}, nil)
	}

	savePath := filepath.Join(uploadDir, file.Filename)
	if err := c.SaveFile(file, savePath); err != nil {
		return "", util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: fmt.Sprintf("cannot save %s file", fieldName),
		}, err)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	var content string
	switch ext {
	case ".pdf":
		content, err = util.ExtractPDFOCR(savePath)
	default:
		return "", util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: fmt.Sprintf("unsupported %s file type", fieldName),
		}, nil)
	}

	if err != nil {
		return "", util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: fmt.Sprintf("failed to extract %s text", fieldName),
		}, err)
	}

	return content, nil
}

func (h *EvaluateHandler) Result(c *fiber.Ctx) error {
	id := c.Params("id")
	job, err := h.uc.GetResult(id)
	if err != nil {
		return util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: "job not found",
		}, nil)
	}
	data := dto.EvaluationTaskDTO{
		ID:              job.ID,
		Status:          job.Status,
		CvMatchRate:     job.CvMatchRate,
		CvFeedback:      job.CvFeedback,
		ProjectScore:    job.ProjectScore,
		ProjectFeedback: job.ProjectFeedback,
		OverallSummary:  job.OverallSummary,
		Breakdown:       job.Breakdown,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}
	return util.SuccessResponse(c, util.SuccessResponseFormat{
		Message: "Success get evaluation result",
		Data:    data,
	})
}

func (h *EvaluateHandler) Test(c *fiber.Ctx) error {
	gemini, err := h.uc.Test()
	if err != nil {
		return util.ErrorResponse(c, util.ErrorResponseFormat{
			Message: "failed to test gemini",
		}, err)
	}
	return util.SuccessResponse(c, util.SuccessResponseFormat{
		Message: "Success test",
		Data:    gemini,
	})
}
