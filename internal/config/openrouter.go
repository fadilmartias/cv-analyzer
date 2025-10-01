package config

import (
	"os"
	"sync"
)

type OpenRouterConfig struct {
	APIKey string
}

var (
	openRouterConfig *OpenRouterConfig
	openRouterOnce   sync.Once
)

func LoadOpenRouterConfig() *OpenRouterConfig {
	openRouterOnce.Do(func() {
		openRouterConfig = &OpenRouterConfig{
			APIKey: os.Getenv("OPENROUTER_API_KEY"),
		}
	})
	return openRouterConfig
}
