package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type EngineSpecState struct {
	epoch      int64
	generation int64
	instanceId uuid.UUID
}

type EngineSpec struct {
	engineId uuid.UUID
	state    *EngineSpecState
	config   string
	enabled  bool
}

type EngineService struct {
	spec   *EngineSpec
	client *EngineHttpClient
	logger *logrus.Entry
}

func NewService(engineId uuid.UUID, client *EngineHttpClient, logger *logrus.Logger) *EngineService {
	return &EngineService{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New()},
		},
		client: client,
		logger: logger.WithField("service", "EngineService"),
	}
}

func (svc *EngineService) Register(ctx context.Context) error {
	epoch, err := svc.client.RegisterInstance(ctx, svc.spec.engineId, svc.spec.state.instanceId)
	if err != nil {
		return fmt.Errorf("register instance: %w", err)
	}
	svc.logger.Printf("Registered instance with epoch %d", epoch)

	svc.spec.state.epoch = epoch
	return nil
}
