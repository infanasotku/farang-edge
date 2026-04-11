package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/infanasotku/farang-edge/internal/config"
	"github.com/infanasotku/farang-edge/internal/controlapi"
	"github.com/infanasotku/farang-edge/internal/engine"
	"github.com/infanasotku/farang-edge/internal/heartbeat"
	"github.com/infanasotku/farang-edge/internal/specsync"
	"github.com/infanasotku/farang-edge/internal/xray"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

type App struct {
	logger *logrus.Logger
	config *config.Config
	svc    *engine.Service
}

func New() (*App, error) {
	logger := config.NewLogger()
	config, err := config.GetAppConfig()
	if err != nil {
		return nil, fmt.Errorf("get app config: %w", err)
	}
	logger.Printf("Config is loaded successfully! ControlBaseUrl: %s, EngineId: %s", config.ControlBaseUrl, config.EngineId)

	httpClient := &http.Client{Timeout: 10 * time.Second}
	client := controlapi.New(config.ControlBaseUrl, config.ControlAuthToken, httpClient)
	xrayEngine := xray.New()
	svc := engine.New(config.EngineId, client, xrayEngine, logger)

	a := App{logger: logger, config: config, svc: svc}
	logger.Println("App is created successfully!")

	return &a, nil
}

func (app *App) Run(ctx context.Context) error {
	err := app.register(ctx)
	if err != nil {
		return fmt.Errorf("register: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return heartbeat.Start(ctx, app.svc, app.logger)
	})
	g.Go(func() error {
		return specsync.Start(ctx, app.svc, app.logger)
	})

	return g.Wait()
}

func (app *App) register(ctx context.Context) error {
	app.logger.Println("Registrating the engine in control plane...")
	err := app.svc.Register(ctx)
	if err != nil {
		return err
	}
	return nil
}
