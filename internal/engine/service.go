package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopped  Status = "stopped"
)

type EngineSpecState struct {
	epoch      int64
	generation int64
	instanceId uuid.UUID
	seq_no     int64
	phase      Status
}

type EngineSpec struct {
	engineId   uuid.UUID
	state      *EngineSpecState
	config     string
	configHash string
	enabled    bool
}

type Service struct {
	spec    *EngineSpec
	control ControlPlane
	engine  Engine
	logger  *logrus.Entry
}

func New(engineId uuid.UUID, control ControlPlane, engine Engine, logger *logrus.Logger) *Service {
	return &Service{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New(), seq_no: 1, phase: StatusStarting},
		},
		control: control,
		engine:  engine,
		logger:  logger.WithField("service", "EngineService"),
	}
}

func (svc *Service) Register(ctx context.Context) error {
	epoch, err := svc.control.RegisterInstance(ctx, svc.spec.engineId, svc.spec.state.instanceId)
	if err != nil {
		return fmt.Errorf("register instance: %w", err)
	}
	svc.logger.Printf("Registered instance %s with epoch %d", svc.spec.state.instanceId, epoch)

	svc.spec.state.epoch = epoch
	return nil
}

func (svc *Service) SendHeartbeat(ctx context.Context) error {
	req := HeartbeatRequest{
		EngineID:   svc.spec.engineId,
		InstanceID: svc.spec.state.instanceId,
		Epoch:      svc.spec.state.epoch,
		SeqNo:      svc.spec.state.seq_no,
		Phase:      svc.spec.state.phase,
		Generation: svc.spec.state.generation,
	}
	err := svc.control.SendHeartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}

	svc.logger.Printf(
		"Sent heartbeat with epoch %d, seq_no %d, phase %s, generation %d",
		svc.spec.state.epoch,
		svc.spec.state.seq_no,
		string(svc.spec.state.phase),
		svc.spec.state.generation,
	)
	svc.spec.state.seq_no += 1
	return nil
}

func (svc *Service) LoadSpec(ctx context.Context) error {
	specResp, err := svc.control.GetSpec(ctx, svc.spec.engineId)
	if err != nil {
		return fmt.Errorf("get spec: %w", err)
	}

	if specResp.Generation != svc.spec.state.generation {
		svc.logger.Printf(
			"Spec generation changed from %d to %d, syncing spec...",
			svc.spec.state.generation,
			specResp.Generation,
		)
		svc.syncState(&specResp)
	}

	return nil
}

func (svc *Service) syncState(snapshot *SpecSnapshot) error {
	configChanged := snapshot.ConfigHash != svc.spec.configHash

	svc.spec.config = snapshot.Config
	svc.spec.configHash = snapshot.ConfigHash
	svc.spec.enabled = snapshot.Enabled
	svc.spec.state.generation = snapshot.Generation

	if configChanged {
		svc.logger.Printf("Config hash changed, updating engine config...")
		err := svc.engine.SetConfig(snapshot.Config)
		if err != nil {
			return fmt.Errorf("set engine config: %w", err)
		}
	}

	if svc.engine.IsEnabled() && !snapshot.Enabled {
		svc.logger.Printf("Disabling engine due to spec change...")
		err := svc.engine.Disable()
		if err != nil {
			return fmt.Errorf("disable engine: %w", err)
		}
	} else if !svc.engine.IsEnabled() && snapshot.Enabled {
		svc.logger.Printf("Enabling engine due to spec change...")
		err := svc.engine.Enable()
		if err != nil {
			return fmt.Errorf("enable engine: %w", err)
		}
	} else if configChanged && snapshot.Enabled {
		svc.logger.Printf("Restarting engine due to config change...")
		err := svc.engine.Disable()
		if err != nil {
			return fmt.Errorf("disable engine: %w", err)
		}
		err = svc.engine.Enable()
		if err != nil {
			return fmt.Errorf("enable engine: %w", err)
		}
	}

	svc.logger.Printf("Spec is synced with generation %d, enabled: %t", snapshot.Generation, snapshot.Enabled)

	return nil
}
