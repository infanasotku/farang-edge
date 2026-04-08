package app

import (
	"context"

	"github.com/google/uuid"
	"github.com/infanasotku/farang-edge/internal/config"
	"github.com/infanasotku/farang-edge/internal/heartbeat"
	"github.com/infanasotku/farang-edge/internal/service"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type App struct {
	logger *logrus.Logger
}

func New() (*App, error) {
	logger := config.NewLogger()

	a := App{logger: logger}
	logger.Println("App is created successfully!")

	return &a, nil
}

func (app *App) Run(ctx context.Context) error {
	svc := service.NewEngineService(uuid.New())

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return heartbeat.Start(ctx, svc, app.logger)
	})

	return g.Wait()
}
