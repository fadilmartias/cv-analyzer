package config

import (
	"os"
	"sync"
)

type GeminiConfig struct {
	APIKey string
}

var (
	geminiConfig *GeminiConfig
	geminiOnce   sync.Once
)

func LoadGeminiConfig() *GeminiConfig {
	geminiOnce.Do(func() {
		geminiConfig = &GeminiConfig{
			APIKey: os.Getenv("GEMINI_API_KEY"),
		}
	})
	return geminiConfig
}
