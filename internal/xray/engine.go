package xray

import (
	"fmt"
	"strings"
	"sync"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all" // Important for loading xray engine properly
)

type Engine struct {
	config     *core.Config
	configHash string
	instance   *core.Instance
	mu         sync.Mutex
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) Apply(config string, configHash string, enabled bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	configChanged := configHash != e.configHash
	if configChanged {
		e.configHash = configHash
		err := e.setConfig(config)
		if err != nil {
			return fmt.Errorf("set config: %w", err)
		}
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

func (e *Engine) IsAlive() bool {
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
