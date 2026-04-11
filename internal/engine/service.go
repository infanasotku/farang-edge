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
	engineId uuid.UUID
	state    *EngineSpecState
	config   map[string]interface{}
	enabled  bool
}

type EngineService struct {
	spec    *EngineSpec
	control ControlPlane
	logger  *logrus.Entry
}

func New(engineId uuid.UUID, control ControlPlane, logger *logrus.Logger) *EngineService {
	return &EngineService{
		spec: &EngineSpec{
			engineId: engineId,
			state:    &EngineSpecState{instanceId: uuid.New(), seq_no: 1, phase: StatusStarting},
		},
		control: control,
		logger:  logger.WithField("service", "EngineService"),
	}
}

func (svc *EngineService) Register(ctx context.Context) error {
	epoch, err := svc.control.RegisterInstance(ctx, svc.spec.engineId, svc.spec.state.instanceId)
	if err != nil {
		return fmt.Errorf("register instance: %w", err)
	}
	svc.logger.Printf("Registered instance %s with epoch %d", svc.spec.state.instanceId, epoch)

	svc.spec.state.epoch = epoch
	return nil
}

func (svc *EngineService) SendHeartbeat(ctx context.Context) error {
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

func (svc *EngineService) LoadSpec(ctx context.Context) error {
	specResp, err := svc.control.GetSpec(ctx, svc.spec.engineId)
	if err != nil {
		return fmt.Errorf("get spec: %w", err)
	}

	if specResp.Generation != svc.spec.state.generation {
		svc.spec.config = specResp.Config
		svc.spec.enabled = specResp.Enabled
		svc.spec.state.generation = specResp.Generation
	}

	return nil
}
