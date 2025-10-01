package config

import (
	"log"
	"os"
	"sync"
)

type AppConfig struct {
	Name    string
	Env     string
	Port    string
	BaseURL string
}

var (
	appConfig *AppConfig
	appOnce   sync.Once
)

func LoadAppConfig() *AppConfig {
	appOnce.Do(func() {
		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "development"
			log.Printf("Warning: APP_ENV not set, defaulting to %s", env)
		}
		appConfig = &AppConfig{
			Name:    os.Getenv("APP_NAME"),
			Env:     env,
			Port:    os.Getenv("APP_PORT"),
			BaseURL: os.Getenv("APP_URL"),
		}
	})
	return appConfig
}
