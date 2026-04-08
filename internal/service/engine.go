package service

import "github.com/google/uuid"

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
	spec *EngineSpec
}

func NewEngineService(engineId uuid.UUID) *EngineService {
	return &EngineService{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New()},
		},
	}
}
