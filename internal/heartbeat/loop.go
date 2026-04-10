package heartbeat

import (
	"context"
	"time"

	"github.com/infanasotku/farang-edge/internal/engine"
	"github.com/sirupsen/logrus"
)

var HEARTBEAT_INTERVAL = 15

func Start(ctx context.Context, svc *engine.EngineService, logger *logrus.Logger) error {
	logger.Println("Registrating the engine in control plane...")
	err := svc.Register(ctx)
	if err != nil {
		return err
	}
	logger.Println("Starting the heartbeat loop...")

	process := func(ctx context.Context) error {
		logger.Println("Sending heartbeat...")
		return svc.SendHeartbeat(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("Cancelling the heartbeat loop...")
			return nil
		case <-time.After(time.Duration(HEARTBEAT_INTERVAL) * time.Second):
			if err := process(ctx); err != nil {
				return err
			}

		}
	}
}
