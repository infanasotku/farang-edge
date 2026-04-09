package engine

import (
	"context"

	"github.com/google/uuid"
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
}

func NewService(engineId uuid.UUID, client *EngineHttpClient) *EngineService {
	return &EngineService{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New()},
		},
		client: client,
	}
}

func (svc *EngineService) Register(ctx context.Context) error {
	return nil
}
