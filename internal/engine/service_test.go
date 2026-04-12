package engine

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type stubEngine struct {
	alive bool
}

func (e *stubEngine) Apply(config string, configHash string, enabled bool) error {
	return nil
}

func (e *stubEngine) IsAlive(ctx context.Context) bool {
	return e.alive
}

type stubBuilder struct{}

func (b *stubBuilder) Build(remoteConfig string, remoteHash string) (BuildResult, error) {
	return BuildResult{Config: remoteConfig, Hash: remoteHash}, nil
}

func TestGetPhaseStartingWhileSyncingEnabledSpec(t *testing.T) {
	svc := New(uuid.New(), nil, &stubEngine{alive: false}, &stubBuilder{}, logrus.New())
	svc.spec.state.generation = 3
	svc.spec.state.syncing = true
	svc.spec.state.pendingEnabled = true

	if got := svc.getPhase(context.Background()); got != StatusStarting {
		t.Fatalf("getPhase() = %q, want %q", got, StatusStarting)
	}
}

func TestGetPhaseFailedWhenEnabledButNotAlive(t *testing.T) {
	svc := New(uuid.New(), nil, &stubEngine{alive: false}, &stubBuilder{}, logrus.New())
	svc.spec.state.generation = 3
	svc.spec.enabled = true

	if got := svc.getPhase(context.Background()); got != StatusFailed {
		t.Fatalf("getPhase() = %q, want %q", got, StatusFailed)
	}
}
