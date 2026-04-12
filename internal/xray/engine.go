package xray

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	observatorypb "github.com/xtls/xray-core/app/observatory/command"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all" // Important for loading xray engine properly
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type RuntimeConfig struct {
	APIListen         string
	ProbeURL          string
	ProbeInterval     time.Duration
	SubjectSelectors  []string
	EnableConcurrency bool
}

type Engine struct {
	runtime    RuntimeConfig
	config     *core.Config
	configHash string
	instance   *core.Instance
	mu         sync.Mutex
}

func New(runtime RuntimeConfig) *Engine {
	return &Engine{runtime: runtime}
}

func (e *Engine) Apply(config string, configHash string, enabled bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	configChanged := configHash != e.configHash
	if configChanged {
		err := e.setConfig(config)
		if err != nil {
			return fmt.Errorf("set config: %w", err)
		}
		e.configHash = configHash
	}

	if enabled {
		if configChanged && e.isEnabled() {
			err := e.ensureDisabled()
			if err != nil {
				return fmt.Errorf("disable engine: %w", err)
			}
		}

		return e.ensureEnabled()
	} else {
		return e.ensureDisabled()
	}
}

func (e *Engine) IsAlive(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(
		e.runtime.APIListen,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return false
	}
	defer conn.Close()

	client := observatorypb.NewObservatoryServiceClient(conn)
	resp, err := client.GetOutboundStatus(ctx, &observatorypb.GetOutboundStatusRequest{})
	if err != nil {
		return false
	}

	for _, status := range resp.Status.Status {
		if !status.Alive {
			return false
		}
	}

	return true
}

func (e *Engine) setConfig(config string) error {
	conf, err := core.LoadConfig("json", strings.NewReader(config))
	if err != nil {
		return err
	}
	e.config = conf
	return nil
}

func (e *Engine) ensureEnabled() error {
	if e.config == nil {
		return fmt.Errorf("config is not set")
	}

	if e.isEnabled() {
		return nil
	}

	engine, err := core.New(e.config)
	if err != nil {
		return err
	}
	err = engine.Start()
	if err != nil {
		return err
	}

	e.instance = engine
	return nil
}

func (e *Engine) ensureDisabled() error {
	if !e.isEnabled() {
		return nil
	}

	err := e.instance.Close()
	if err != nil {
		return err
	}
	e.instance = nil
	return nil
}

func (e *Engine) isEnabled() bool {
	return e.instance != nil
}
