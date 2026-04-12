package engine

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type registerCall struct {
	engineID   uuid.UUID
	instanceID uuid.UUID
}

type applyCall struct {
	config     string
	configHash string
	enabled    bool
}

type fakeControlPlane struct {
	registerEpoch int64
	registerErr   error
	registerCalls []registerCall

	heartbeatErr   error
	heartbeatCalls []HeartbeatRequest

	spec    SpecSnapshot
	specErr error
}

func (c *fakeControlPlane) RegisterInstance(ctx context.Context, engineID, instanceID uuid.UUID) (int64, error) {
	c.registerCalls = append(c.registerCalls, registerCall{engineID: engineID, instanceID: instanceID})
	if c.registerErr != nil {
		return 0, c.registerErr
	}
	return c.registerEpoch, nil
}

func (c *fakeControlPlane) SendHeartbeat(ctx context.Context, req HeartbeatRequest) error {
	c.heartbeatCalls = append(c.heartbeatCalls, req)
	return c.heartbeatErr
}

func (c *fakeControlPlane) GetSpec(ctx context.Context, engineID uuid.UUID) (SpecSnapshot, error) {
	if c.specErr != nil {
		return SpecSnapshot{}, c.specErr
	}
	return c.spec, nil
}

type fakeEngine struct {
	alive      bool
	applyErrs  []error
	applyCalls []applyCall
}

func (e *fakeEngine) Apply(config string, configHash string, enabled bool) error {
	e.applyCalls = append(e.applyCalls, applyCall{
		config:     config,
		configHash: configHash,
		enabled:    enabled,
	})

	if len(e.applyErrs) == 0 {
		return nil
	}

	err := e.applyErrs[0]
	e.applyErrs = e.applyErrs[1:]
	return err
}

func (e *fakeEngine) IsAlive(ctx context.Context) bool {
	return e.alive
}

type fakeBuilder struct {
	result     BuildResult
	err        error
	buildCalls int
	lastConfig string
	lastHash   string
}

func (b *fakeBuilder) Build(remoteConfig string, remoteHash string) (BuildResult, error) {
	b.buildCalls++
	b.lastConfig = remoteConfig
	b.lastHash = remoteHash
	if b.err != nil {
		return BuildResult{}, b.err
	}
	return b.result, nil
}

func newTestService(control *fakeControlPlane, eng *fakeEngine, builder *fakeBuilder) *Service {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return New(uuid.New(), control, eng, builder, logger)
}

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		control := &fakeControlPlane{registerEpoch: 42}
		svc := newTestService(control, &fakeEngine{}, &fakeBuilder{})

		if err := svc.Register(context.Background()); err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		if got := svc.spec.state.epoch; got != 42 {
			t.Fatalf("epoch = %d, want 42", got)
		}
		if len(control.registerCalls) != 1 {
			t.Fatalf("register calls = %d, want 1", len(control.registerCalls))
		}
		if control.registerCalls[0].engineID != svc.spec.engineId {
			t.Fatalf("engineID = %s, want %s", control.registerCalls[0].engineID, svc.spec.engineId)
		}
		if control.registerCalls[0].instanceID != svc.spec.state.instanceId {
			t.Fatalf("instanceID = %s, want %s", control.registerCalls[0].instanceID, svc.spec.state.instanceId)
		}
	})

	t.Run("already registered", func(t *testing.T) {
		control := &fakeControlPlane{}
		svc := newTestService(control, &fakeEngine{}, &fakeBuilder{})
		svc.spec.state.epoch = 10

		err := svc.Register(context.Background())
		if err == nil || !strings.Contains(err.Error(), "already registered") {
			t.Fatalf("Register() error = %v, want already registered", err)
		}
		if len(control.registerCalls) != 0 {
			t.Fatalf("register calls = %d, want 0", len(control.registerCalls))
		}
	})

	t.Run("control plane error", func(t *testing.T) {
		control := &fakeControlPlane{registerErr: errors.New("boom")}
		svc := newTestService(control, &fakeEngine{}, &fakeBuilder{})

		err := svc.Register(context.Background())
		if err == nil || !strings.Contains(err.Error(), "register instance: boom") {
			t.Fatalf("Register() error = %v, want wrapped control plane error", err)
		}
		if got := svc.spec.state.epoch; got != 0 {
			t.Fatalf("epoch = %d, want 0", got)
		}
	})
}

func TestSendHeartbeat(t *testing.T) {
	t.Run("success increments seq and sends request", func(t *testing.T) {
		control := &fakeControlPlane{}
		engine := &fakeEngine{alive: true}
		svc := newTestService(control, engine, &fakeBuilder{})
		svc.spec.state.epoch = 7
		svc.spec.state.generation = 3
		svc.spec.enabled = true

		if err := svc.SendHeartbeat(context.Background()); err != nil {
			t.Fatalf("SendHeartbeat() error = %v", err)
		}

		if got := svc.spec.state.seq_no; got != 2 {
			t.Fatalf("seq_no = %d, want 2", got)
		}
		if len(control.heartbeatCalls) != 1 {
			t.Fatalf("heartbeat calls = %d, want 1", len(control.heartbeatCalls))
		}

		req := control.heartbeatCalls[0]
		if req.Epoch != 7 {
			t.Fatalf("req.Epoch = %d, want 7", req.Epoch)
		}
		if req.SeqNo != 1 {
			t.Fatalf("req.SeqNo = %d, want 1", req.SeqNo)
		}
		if req.Generation != 3 {
			t.Fatalf("req.Generation = %d, want 3", req.Generation)
		}
		if req.Phase != StatusRunning {
			t.Fatalf("req.Phase = %q, want %q", req.Phase, StatusRunning)
		}
	})

	t.Run("error does not increment seq", func(t *testing.T) {
		control := &fakeControlPlane{heartbeatErr: errors.New("timeout")}
		svc := newTestService(control, &fakeEngine{}, &fakeBuilder{})

		err := svc.SendHeartbeat(context.Background())
		if err != nil {
			t.Fatalf("SendHeartbeat() error = %v, want nil", err)
		}
		if got := svc.spec.state.seq_no; got != 1 {
			t.Fatalf("seq_no = %d, want 1", got)
		}
	})
}

func TestLoadSpec(t *testing.T) {
	t.Run("get spec error", func(t *testing.T) {
		control := &fakeControlPlane{specErr: errors.New("unavailable")}
		svc := newTestService(control, &fakeEngine{}, &fakeBuilder{})

		err := svc.LoadSpec(context.Background())
		if err == nil || !strings.Contains(err.Error(), "get spec: unavailable") {
			t.Fatalf("LoadSpec() error = %v, want wrapped get spec error", err)
		}
	})

	t.Run("same generation no-op", func(t *testing.T) {
		control := &fakeControlPlane{spec: SpecSnapshot{Generation: 2}}
		builder := &fakeBuilder{}
		engine := &fakeEngine{}
		svc := newTestService(control, engine, builder)
		svc.spec.state.generation = 2

		if err := svc.LoadSpec(context.Background()); err != nil {
			t.Fatalf("LoadSpec() error = %v", err)
		}
		if builder.buildCalls != 0 {
			t.Fatalf("build calls = %d, want 0", builder.buildCalls)
		}
		if len(engine.applyCalls) != 0 {
			t.Fatalf("apply calls = %d, want 0", len(engine.applyCalls))
		}
	})

	t.Run("generation change applies effective config", func(t *testing.T) {
		control := &fakeControlPlane{spec: SpecSnapshot{
			Config:     "remote-config",
			ConfigHash: "remote-hash",
			Enabled:    true,
			Generation: 5,
		}}
		builder := &fakeBuilder{result: BuildResult{Config: "effective-config", Hash: "effective-hash"}}
		engine := &fakeEngine{}
		svc := newTestService(control, engine, builder)

		if err := svc.LoadSpec(context.Background()); err != nil {
			t.Fatalf("LoadSpec() error = %v", err)
		}

		if builder.buildCalls != 1 {
			t.Fatalf("build calls = %d, want 1", builder.buildCalls)
		}
		if builder.lastConfig != "remote-config" || builder.lastHash != "remote-hash" {
			t.Fatalf("builder inputs = (%q, %q), want (%q, %q)", builder.lastConfig, builder.lastHash, "remote-config", "remote-hash")
		}
		if len(engine.applyCalls) != 1 {
			t.Fatalf("apply calls = %d, want 1", len(engine.applyCalls))
		}
		if engine.applyCalls[0] != (applyCall{config: "effective-config", configHash: "effective-hash", enabled: true}) {
			t.Fatalf("apply call = %#v, want effective config", engine.applyCalls[0])
		}
		if svc.spec.config != "effective-config" || svc.spec.configHash != "effective-hash" {
			t.Fatalf("stored effective config = (%q, %q), want (%q, %q)", svc.spec.config, svc.spec.configHash, "effective-config", "effective-hash")
		}
		if !svc.spec.enabled {
			t.Fatalf("enabled = false, want true")
		}
		if svc.spec.state.generation != 5 {
			t.Fatalf("generation = %d, want 5", svc.spec.state.generation)
		}
		if svc.spec.state.maxSeenGeneration != 5 {
			t.Fatalf("maxSeenGeneration = %d, want 5", svc.spec.state.maxSeenGeneration)
		}
	})

	t.Run("build error keeps previous state", func(t *testing.T) {
		control := &fakeControlPlane{spec: SpecSnapshot{
			Config:     "remote-config",
			ConfigHash: "remote-hash",
			Enabled:    true,
			Generation: 7,
		}}
		builder := &fakeBuilder{err: errors.New("bad overlay")}
		engine := &fakeEngine{}
		svc := newTestService(control, engine, builder)
		svc.spec.config = "old-config"
		svc.spec.configHash = "old-hash"
		svc.spec.enabled = false
		svc.spec.state.generation = 3

		if err := svc.LoadSpec(context.Background()); err != nil {
			t.Fatalf("LoadSpec() error = %v, want nil on build error", err)
		}

		if len(engine.applyCalls) != 0 {
			t.Fatalf("apply calls = %d, want 0", len(engine.applyCalls))
		}
		if svc.spec.config != "old-config" || svc.spec.configHash != "old-hash" || svc.spec.enabled {
			t.Fatalf("state changed unexpectedly after build error")
		}
		if svc.spec.state.generation != 3 {
			t.Fatalf("generation = %d, want 3", svc.spec.state.generation)
		}
		if svc.spec.state.maxSeenGeneration != 7 {
			t.Fatalf("maxSeenGeneration = %d, want 7", svc.spec.state.maxSeenGeneration)
		}
	})

	t.Run("apply error triggers rollback", func(t *testing.T) {
		control := &fakeControlPlane{spec: SpecSnapshot{
			Config:     "remote-config",
			ConfigHash: "remote-hash",
			Enabled:    true,
			Generation: 8,
		}}
		builder := &fakeBuilder{result: BuildResult{Config: "effective-config", Hash: "effective-hash"}}
		engine := &fakeEngine{applyErrs: []error{errors.New("apply failed"), nil}}
		svc := newTestService(control, engine, builder)
		svc.spec.config = "old-config"
		svc.spec.configHash = "old-hash"
		svc.spec.enabled = false
		svc.spec.state.generation = 4

		if err := svc.LoadSpec(context.Background()); err != nil {
			t.Fatalf("LoadSpec() error = %v, want nil when rollback succeeds", err)
		}

		if len(engine.applyCalls) != 2 {
			t.Fatalf("apply calls = %d, want 2", len(engine.applyCalls))
		}
		if engine.applyCalls[1] != (applyCall{config: "old-config", configHash: "old-hash", enabled: false}) {
			t.Fatalf("rollback apply call = %#v, want previous config", engine.applyCalls[1])
		}
		if svc.spec.config != "old-config" || svc.spec.configHash != "old-hash" || svc.spec.enabled {
			t.Fatalf("state changed unexpectedly after successful rollback")
		}
		if svc.spec.state.generation != 4 {
			t.Fatalf("generation = %d, want 4", svc.spec.state.generation)
		}
		if svc.spec.state.maxSeenGeneration != 8 {
			t.Fatalf("maxSeenGeneration = %d, want 8", svc.spec.state.maxSeenGeneration)
		}
	})

	t.Run("rollback error returns error", func(t *testing.T) {
		control := &fakeControlPlane{spec: SpecSnapshot{
			Config:     "remote-config",
			ConfigHash: "remote-hash",
			Enabled:    true,
			Generation: 9,
		}}
		builder := &fakeBuilder{result: BuildResult{Config: "effective-config", Hash: "effective-hash"}}
		engine := &fakeEngine{applyErrs: []error{errors.New("apply failed"), errors.New("rollback failed")}}
		svc := newTestService(control, engine, builder)
		svc.spec.config = "old-config"
		svc.spec.configHash = "old-hash"
		svc.spec.enabled = true
		svc.spec.state.generation = 4

		err := svc.LoadSpec(context.Background())
		if err == nil || !strings.Contains(err.Error(), "failed to rollback engine config: rollback failed") {
			t.Fatalf("LoadSpec() error = %v, want wrapped rollback error", err)
		}
		if svc.spec.state.generation != 4 {
			t.Fatalf("generation = %d, want 4", svc.spec.state.generation)
		}
		if svc.spec.state.maxSeenGeneration != 9 {
			t.Fatalf("maxSeenGeneration = %d, want 9", svc.spec.state.maxSeenGeneration)
		}
	})
}

func TestGetPhaseLocked(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(*Service, *fakeEngine)
		expect Status
	}{
		{
			name: "starting when generation is zero",
			setup: func(svc *Service, eng *fakeEngine) {
				svc.spec.state.generation = 0
			},
			expect: StatusStarting,
		},
		{
			name: "rolledback when max seen generation is ahead",
			setup: func(svc *Service, eng *fakeEngine) {
				svc.spec.state.generation = 3
				svc.spec.state.maxSeenGeneration = 4
			},
			expect: StatusRolledback,
		},
		{
			name: "running when enabled and alive",
			setup: func(svc *Service, eng *fakeEngine) {
				svc.spec.state.generation = 3
				svc.spec.enabled = true
				eng.alive = true
			},
			expect: StatusRunning,
		},
		{
			name: "failed when enabled and not alive",
			setup: func(svc *Service, eng *fakeEngine) {
				svc.spec.state.generation = 3
				svc.spec.enabled = true
				eng.alive = false
			},
			expect: StatusFailed,
		},
		{
			name: "idle when disabled",
			setup: func(svc *Service, eng *fakeEngine) {
				svc.spec.state.generation = 3
				svc.spec.enabled = false
			},
			expect: StatusIdle,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			engine := &fakeEngine{}
			svc := newTestService(&fakeControlPlane{}, engine, &fakeBuilder{})
			tc.setup(svc, engine)

			if got := svc.getPhaseLocked(context.Background()); got != tc.expect {
				t.Fatalf("getPhaseLocked() = %q, want %q", got, tc.expect)
			}
		})
	}
}
