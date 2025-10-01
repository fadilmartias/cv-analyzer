package util

import (
	"fmt"
	"runtime/debug"

	"github.com/fadilmartias/cv-analyzer/internal/config"
	"github.com/fadilmartias/cv-analyzer/internal/response"
	"github.com/gofiber/fiber/v2"
)

type SuccessResponseFormat struct {
	Code       int
	Message    string
	Data       any
	Pagination *response.Pagination
	Meta       any
}

type OrderedSuccessResponse struct {
	Success    bool                 `json:"success"`
	Message    string               `json:"message"`
	Meta       any                  `json:"meta,omitempty"`
	Pagination *response.Pagination `json:"pagination,omitempty"`
	Data       any                  `json:"data,omitempty"`
}

type ErrorResponseFormat struct {
	Code       int
	Message    string
	DevMessage string
	Details    any
	Trace      string
}

type OrderedErrorResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DevMessage string `json:"dev_message,omitempty"`
	Details    any    `json:"details,omitempty"`
	Trace      string `json:"trace,omitempty"`
}

type FormError struct {
	Errors  map[string]string
	Message string
}

func (e *FormError) Error() string {
	return fmt.Sprintf("form error: %s", e.Message)
}

func NewFormError(message string, errors map[string]string) *FormError {
	return &FormError{
		Message: message,
		Errors:  errors,
	}
}

// SuccessResponse mengirim response JSON standar untuk sukses
func SuccessResponse(c *fiber.Ctx, params SuccessResponseFormat) error {
	response := OrderedSuccessResponse{
		Success:    true,
		Message:    params.Message,
		Data:       params.Data,
		Pagination: params.Pagination,
		Meta:       params.Meta,
	}
	return c.Status(params.Code).JSON(response)
}

// ErrorResponse mengirim response JSON standar untuk error
func ErrorResponse(c *fiber.Ctx, params ErrorResponseFormat, errs ...error) error {
	response := OrderedErrorResponse{
		Success: false,
		Message: params.Message,
	}
	if params.Details != nil {
		response.Details = params.Details
	}
	if config.LoadAppConfig().Env != "production" {
		if len(errs) > 0 && errs[0] != nil {
			response.DevMessage = errs[0].Error()
			response.Details = errs[0]
			response.Trace = string(debug.Stack())
		}

		if params.DevMessage != "" {
			response.DevMessage = params.DevMessage
		}
		if params.Details != nil {
			response.Details = params.Details
		}
		if params.Trace != "" {
			response.Trace = params.Trace
		}
	}

	errorCode := params.Code
	if params.Code == 0 {
		errorCode = fiber.StatusInternalServerError
	}
	return c.Status(errorCode).JSON(response)
}
