package heartbeat

import (
	"context"

	"github.com/infanasotku/farang-edge/internal/engine"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, svc *engine.EngineService, logger *logrus.Logger) error {
	logger.Println("Registrating the engine in control plane...")
	err := svc.Register(ctx)
	if err != nil {
		return err
	}
	logger.Println("Starting the heartbeat loop...")

	process := func(ctx context.Context) {
		// logger.Println("Sending heartbeat...")
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("Cancelling the heartbeat loop...")
			return nil
		default:
			process(ctx)
		}
	}
}
