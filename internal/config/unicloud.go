package config

import (
	"os"
	"sync"
)

type UniCloudConfig struct {
	APIKey string
}

var (
	uniCloudConfig *UniCloudConfig
	uniCloudOnce   sync.Once
)

func LoadUniCloudConfig() *UniCloudConfig {
	uniCloudOnce.Do(func() {
		uniCloudConfig = &UniCloudConfig{
			APIKey: os.Getenv("UNICLOUD_API_KEY"),
		}
	})
	return uniCloudConfig
}
