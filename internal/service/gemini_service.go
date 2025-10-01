package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/fadilmartias/cv-analyzer/internal/config"
	"google.golang.org/genai"
)

type GeminiServiceInterface interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	GenerateContent(ctx context.Context, model string, prompt string) (*genai.GenerateContentResponse, error)
	Test() (string, error)
}

type GeminiService struct {
	Client            *genai.Client
	Ctx               context.Context
	MaxRetries        int
	BaseDelay         time.Duration
	MaxDelay          time.Duration
	RequestTimeout    time.Duration
	consecutiveErrors int
	circuitBreakerMax int
}

func NewGeminiService(ctx context.Context) (*GeminiService, error) {
	geminiConfig := config.LoadGeminiConfig()
	apiKey := geminiConfig.APIKey
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}
	return &GeminiService{
		Client:            client,
		Ctx:               ctx,
		MaxRetries:        3,
		BaseDelay:         time.Second,
		MaxDelay:          90 * time.Second,
		RequestTimeout:    90 * time.Second,
		circuitBreakerMax: 5,
	}, nil
}

func (s *GeminiService) Test() (string, error) {
	result, err := s.Client.Models.GenerateContent(
		s.Ctx,
		"gemini-2.5-flash",
		genai.Text("Explain how AI works in a few words"),
		nil,
	)
	if err != nil {
		return "", err
	}
	return result.Text(), nil
}

func (s *GeminiService) GenerateContent(ctx context.Context, model string, prompt string) (*genai.GenerateContentResponse, error) {
	if model == "" {
		return nil, fmt.Errorf("model name cannot be empty")
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	if s.consecutiveErrors >= s.circuitBreakerMax {
		return nil, fmt.Errorf("circuit breaker open: too many consecutive errors (%d)", s.consecutiveErrors)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, s.RequestTimeout)
	defer cancel()

	var lastErr error
	for attempt := 0; attempt <= s.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := s.calculateBackoff(attempt)
			log.Printf("Retry attempt %d/%d for GenerateContent after %v",
				attempt, s.MaxRetries, delay)

			select {
			case <-time.After(delay):
			case <-timeoutCtx.Done():
				return nil, fmt.Errorf("context timeout during retry: %w", timeoutCtx.Err())
			}
		}

		genConfig := &genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.1)),
		}

		result, err := s.Client.Models.GenerateContent(
			timeoutCtx,
			model,
			genai.Text(prompt),
			genConfig,
		)

		if err == nil {
			s.consecutiveErrors = 0
			if err := s.validateGenerateResponse(result); err != nil {
				return nil, fmt.Errorf("invalid response: %w", err)
			}

			return result, nil
		}

		lastErr = err

		if !s.isRetryableError(err) {
			log.Printf("Non-retryable error: %v", err)
			s.consecutiveErrors++
			return nil, fmt.Errorf("generate content failed: %w", err)
		}

		log.Printf("Retryable error on attempt %d: %v", attempt+1, err)
	}

	s.consecutiveErrors++
	return nil, fmt.Errorf("max retries (%d) exceeded for GenerateContent: %w", s.MaxRetries, lastErr)
}

func (s *GeminiService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		return nil, fmt.Errorf("text for embedding cannot be empty")
	}

	if len(trimmedText) > 10000 {
		log.Printf("Warning: text length %d exceeds recommended limit, truncating...", len(trimmedText))
		trimmedText = trimmedText[:10000]
	}

	if s.consecutiveErrors >= s.circuitBreakerMax {
		return nil, fmt.Errorf("circuit breaker open: too many consecutive errors (%d)", s.consecutiveErrors)
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, s.RequestTimeout)
	defer cancel()

	content := []*genai.Content{genai.NewContentFromText(trimmedText, genai.RoleUser)}

	var lastErr error
	for attempt := 0; attempt <= s.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := s.calculateBackoff(attempt)
			log.Printf("Retry attempt %d/%d for GenerateEmbedding after %v",
				attempt, s.MaxRetries, delay)

			select {
			case <-time.After(delay):
				// Continue to retry
			case <-timeoutCtx.Done():
				return nil, fmt.Errorf("context timeout during retry: %w", timeoutCtx.Err())
			}
		}

		result, err := s.Client.Models.EmbedContent(
			timeoutCtx,
			"gemini-embedding-001",
			content,
			nil,
		)

		if err == nil {
			s.consecutiveErrors = 0
			embeddings, err := s.validateEmbeddingResponse(result)
			if err != nil {
				return nil, fmt.Errorf("invalid embedding response: %w", err)
			}

			return embeddings, nil
		}

		lastErr = err

		if !s.isRetryableError(err) {
			log.Printf("Non-retryable error: %v", err)
			s.consecutiveErrors++
			return nil, fmt.Errorf("generate embedding failed: %w", err)
		}

		log.Printf("Retryable error on attempt %d: %v", attempt+1, err)
	}

	s.consecutiveErrors++
	return nil, fmt.Errorf("max retries (%d) exceeded for GenerateEmbedding: %w", s.MaxRetries, lastErr)
}
func (s *GeminiService) calculateBackoff(attempt int) time.Duration {
	delay := s.BaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))

	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}

	jitter := time.Duration(float64(delay) * 0.25)
	delay = delay - jitter/2 + time.Duration(float64(jitter)*0.5)

	return delay
}

func (s *GeminiService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	if strings.Contains(errMsg, "context canceled") ||
		strings.Contains(errMsg, "context deadline exceeded") {
		return false
	}
	if apiErr, ok := err.(*genai.APIError); ok {
		switch apiErr.Code {
		case 429: // Rate limit
			return true
		case 500, 502, 503, 504: // Server errors
			return true
		case 400, 401, 403, 404: // Client errors
			return false
		}
	}

	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "temporary failure") ||
		strings.Contains(errMsg, "EOF") {
		return true
	}

	return false
}

func (s *GeminiService) validateGenerateResponse(resp *genai.GenerateContentResponse) error {
	if resp == nil {
		return fmt.Errorf("response is nil")
	}

	if len(resp.Candidates) == 0 {
		return fmt.Errorf("no candidates in response")
	}

	if resp.Candidates[0].Content == nil {
		return fmt.Errorf("candidate content is nil")
	}

	if len(resp.Candidates[0].Content.Parts) == 0 {
		return fmt.Errorf("no parts in content")
	}

	return nil
}

func (s *GeminiService) validateEmbeddingResponse(resp *genai.EmbedContentResponse) ([]float32, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	embeddings := resp.Embeddings[0].Values

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("embedding vector is empty")
	}

	for i, val := range embeddings {
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			return nil, fmt.Errorf("invalid embedding value at index %d: %v", i, val)
		}
	}

	return embeddings, nil
}

func (s *GeminiService) ResetCircuitBreaker() {
	s.consecutiveErrors = 0
	log.Println("Circuit breaker reset")
}
func (s *GeminiService) GetCircuitBreakerStatus() (consecutiveErrors int, isOpen bool) {
	return s.consecutiveErrors, s.consecutiveErrors >= s.circuitBreakerMax
}
