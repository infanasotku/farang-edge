package config

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	ControlBaseUrl   string
	ControlAuthToken string
	EngineId         uuid.UUID
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

	controlAuthToken, err := mustEnv("CONTROL_AUTH_TOKEN")
	if err != nil {
		return nil, err
	}

	return &Config{
		ControlBaseUrl:   controlBaseUrl,
		ControlAuthToken: controlAuthToken,
		EngineId:         uuid.MustParse(engineId),
	}, nil
}
