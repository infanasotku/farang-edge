package xray

import (
	"fmt"
	"strings"

	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all" // Important for loading xray engine properly
)

type Engine struct {
	config   *core.Config
	instance *core.Instance
}

func New() *Engine {
	return &Engine{}
}

func (e *Engine) SetConfig(config string) error {
	conf, err := core.LoadConfig("json", strings.NewReader(config))
	if err != nil {
		return err
	}
	e.config = conf
	return nil
}

func (e *Engine) Enable() error {
	if e.config == nil {
		return fmt.Errorf("config is not set")
	}

	if e.instance != nil {
		return fmt.Errorf("engine is already enabled")
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

func (e *Engine) Disable() error {
	if e.instance == nil {
		return fmt.Errorf("engine is not enabled")
	}

	err := e.instance.Close()
	if err != nil {
		return err
	}
	e.instance = nil
	return nil
}

func (e *Engine) IsEnabled() bool {
	return e.instance != nil
}
