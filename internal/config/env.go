package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ControlBaseUrl string
	EngineId       string
}

func mustEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("%s not specified", key)
	}
	return v, nil
}

func GetAppConfig() (*Config, error) {
	_ = godotenv.Load()

	engineId, err := mustEnv("ENGINE_ID")
	if err != nil {
		return nil, err
	}

	controlBaseUrl, err := mustEnv("CONTROL_BASE_URL")
	if err != nil {
		return nil, err
	}

	return &Config{
		ControlBaseUrl: controlBaseUrl,
		EngineId:       engineId,
	}, nil
}
