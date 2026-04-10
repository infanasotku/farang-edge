package app

import (
	"context"
	"fmt"

	"github.com/infanasotku/farang-edge/internal/config"
	"github.com/infanasotku/farang-edge/internal/engine"
	"github.com/infanasotku/farang-edge/internal/heartbeat"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type App struct {
	logger *logrus.Logger
	config *config.Config
}

func New() (*App, error) {
	logger := config.NewLogger()
	config, err := config.GetAppConfig()
	if err != nil {
		return nil, fmt.Errorf("get app config: %w", err)
	}
	logger.Printf("App config: %+v", config)

	a := App{logger: logger, config: config}
	logger.Println("App is created successfully!")

	return &a, nil
}

func (app *App) Run(ctx context.Context) error {
	client := engine.NewClient(app.config.ControlBaseUrl, app.config.ControlAuthToken)
	svc := engine.NewService(app.config.EngineId, client, app.logger)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return heartbeat.Start(ctx, svc, app.logger)
	})

	return g.Wait()
}
