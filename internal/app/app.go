package app

import (
	"context"

	"github.com/infanasotku/farang-edge/internal/config"
	"github.com/sirupsen/logrus"
)

type App struct {
	logger *logrus.Logger
}

func New(ctx context.Context) (*App, error) {
	logger := config.NewLogger()

	a := App{logger: logger}

	logger.Println("App is created successfully!")

	return &a, nil
}
