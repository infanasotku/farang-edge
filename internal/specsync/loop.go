package specsync

import (
	"context"
	"time"

	"github.com/infanasotku/farang-edge/internal/engine"
	"github.com/sirupsen/logrus"
)

var SYNC_INTERVAL = 10

func Start(ctx context.Context, svc *engine.Service, logger *logrus.Logger) error {
	logger.Println("Starting the spec sync loop...")

	process := func(ctx context.Context) error {
		logger.Println("Polling spec...")
		return svc.LoadSpec(ctx)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Println("Cancelling the spec sync loop...")
			return nil
		case <-time.After(time.Duration(SYNC_INTERVAL) * time.Second):
			if err := process(ctx); err != nil {
				return err
			}
		}
	}
}
