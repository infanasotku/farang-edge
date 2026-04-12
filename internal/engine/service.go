package engine

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Status string

const (
	StatusStarting   Status = "starting"
	StatusRunning    Status = "running"
	StatusFailed     Status = "failed"
	StatusIdle       Status = "idle"
	StatusRolledback Status = "rolledback"
)

type EngineSpecState struct {
	epoch             int64
	generation        int64
	maxSeenGeneration int64
	instanceId        uuid.UUID
	seq_no            int64
}

type EngineSpec struct {
	engineId   uuid.UUID
	state      *EngineSpecState
	config     string
	configHash string
	enabled    bool
}

type Service struct {
	spec       *EngineSpec
	control    ControlPlane
	engine     Engine
	cfgBuilder CfgBulder
	logger     *logrus.Entry
	mu         sync.Mutex
}

func New(engineId uuid.UUID, control ControlPlane, engine Engine, cfgBuilder CfgBulder, logger *logrus.Logger) *Service {
	return &Service{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New(), seq_no: 1},
		},
		control:    control,
		engine:     engine,
		cfgBuilder: cfgBuilder,
		logger:     logger.WithField("service", "EngineService"),
	}
}

func (svc *Service) Register(ctx context.Context) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	if svc.spec.state.epoch != 0 {
		return fmt.Errorf("instance is already registered with epoch %d", svc.spec.state.epoch)
	}

	engineID := svc.spec.engineId
	instanceID := svc.spec.state.instanceId

	epoch, err := svc.control.RegisterInstance(ctx, engineID, instanceID)
	if err != nil {
		return fmt.Errorf("register instance: %w", err)
	}
	svc.logger.Printf("Registered instance %s with epoch %d", instanceID, epoch)

	svc.spec.state.epoch = epoch
	return nil
}

func (svc *Service) SendHeartbeat(ctx context.Context) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	req := HeartbeatRequest{
		EngineID:   svc.spec.engineId,
		InstanceID: svc.spec.state.instanceId,
		Epoch:      svc.spec.state.epoch,
		SeqNo:      svc.spec.state.seq_no,
		Generation: svc.spec.state.generation,
	}
	req.Phase = svc.getPhaseLocked(ctx)

	err := svc.control.SendHeartbeat(ctx, req)
	if err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}

	svc.logger.Printf(
		"Sent heartbeat with epoch %d, seq_no %d, phase %s, generation %d",
		req.Epoch,
		req.SeqNo,
		string(req.Phase),
		req.Generation,
	)
	svc.spec.state.seq_no += 1
	return nil
}

func (svc *Service) LoadSpec(ctx context.Context) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	engineID := svc.spec.engineId
	currentGeneration := svc.spec.state.generation

	specResp, err := svc.control.GetSpec(ctx, engineID)
	if err != nil {
		return fmt.Errorf("get spec: %w", err)
	}

	if specResp.Generation != currentGeneration {
		svc.logger.Printf(
			"Spec generation changed from %d to %d, syncing spec...",
			currentGeneration,
			specResp.Generation,
		)
		return svc.syncState(&specResp)
	}

	return nil
}

func (svc *Service) syncState(snapshot *SpecSnapshot) error {
	svc.spec.state.maxSeenGeneration = max(svc.spec.state.maxSeenGeneration, snapshot.Generation)
	prevConfig := svc.spec.config
	prevConfigHash := svc.spec.configHash
	prevEnabled := svc.spec.enabled

	effective, err := svc.cfgBuilder.Build(snapshot.Config, snapshot.ConfigHash)
	if err != nil {
		svc.logger.Errorf("Failed to build effective config: %v, no operation performed", err)
		return nil
	}

	err = svc.engine.Apply(effective.Config, effective.Hash, snapshot.Enabled)
	if err != nil {
		svc.logger.Errorf("Failed to apply new spec: %v, rolling back...", err)
		rollbackErr := svc.engine.Apply(prevConfig, prevConfigHash, prevEnabled)
		if rollbackErr != nil {
			return fmt.Errorf("failed to rollback engine config: %w", rollbackErr)
		}
		svc.logger.Warningf("Successfully rolled back to previous engine config")
		return nil
	}

	svc.spec.config = effective.Config
	svc.spec.configHash = effective.Hash
	svc.spec.enabled = snapshot.Enabled
	svc.spec.state.generation = snapshot.Generation
	svc.logger.Printf("Spec is synced with generation %d, enabled: %t", snapshot.Generation, snapshot.Enabled)

	return nil
}

func (svc *Service) getPhaseLocked(ctx context.Context) Status {
	if svc.spec.state.generation == 0 {
		return StatusStarting
	}

	if svc.spec.state.maxSeenGeneration > svc.spec.state.generation {
		return StatusRolledback
	}

	if svc.spec.enabled {
		alive := svc.engine.IsAlive(ctx)

		if !alive {
			return StatusFailed
		}
		return StatusRunning
	}

	return StatusIdle
}
