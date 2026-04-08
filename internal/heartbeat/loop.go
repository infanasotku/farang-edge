package heartbeat

import (
	"context"

	"github.com/infanasotku/farang-edge/internal/service"
	"github.com/sirupsen/logrus"
)

func Start(ctx context.Context, svc *service.EngineService, logger *logrus.Logger) error {
	logger.Println("Registrating the engine in control plane...")
	// 1. Register the engine in control plane

	// end of registration
	logger.Println("Starting the heartbeat loop...")

	process := func(ctx context.Context) {
		logger.Println("Sending heartbeat...")
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
