package engine

import (
	"context"

	"github.com/google/uuid"
)

type ControlPlane interface {
	RegisterInstance(ctx context.Context, engineID, instanceID uuid.UUID) (int64, error)
	SendHeartbeat(ctx context.Context, req HeartbeatRequest) error
	GetSpec(ctx context.Context, engineID uuid.UUID) (SpecSnapshot, error)
}

type HeartbeatRequest struct {
	EngineID   uuid.UUID
	InstanceID uuid.UUID
	Epoch      int64
	SeqNo      int64
	Phase      Status
	Generation int64
}

type SpecSnapshot struct {
	Config     string
	ConfigHash string
	Enabled    bool
	Generation int64
}

type Engine interface {
	Apply(config string, configHash string, enabled bool) error
	IsAlive() bool
}

type CfgBulder interface {
	Build(remoteConfig string, remoteHash string) (BuildResult, error)
}

type BuildResult struct {
	Config string
	Hash   string
}
