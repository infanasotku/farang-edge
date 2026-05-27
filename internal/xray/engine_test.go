package xray

import (
	"errors"
	"strings"
	"testing"
)

type closeFunc func() error

func (f closeFunc) Close() error {
	return f()
}

func TestEnsureDisabled(t *testing.T) {
	t.Run("recovers close of closed channel panic", func(t *testing.T) {
		engine := &Engine{
			instance: closeFunc(func() error {
				panic("close of closed channel")
			}),
		}

		if err := engine.ensureDisabled(); err != nil {
			t.Fatalf("ensureDisabled() error = %v, want nil", err)
		}
		if engine.instance != nil {
			t.Fatalf("instance = %v, want nil", engine.instance)
		}
	})

	t.Run("returns close error", func(t *testing.T) {
		closeErr := errors.New("close failed")
		engine := &Engine{
			instance: closeFunc(func() error {
				return closeErr
			}),
		}

		if err := engine.ensureDisabled(); !errors.Is(err, closeErr) {
			t.Fatalf("ensureDisabled() error = %v, want %v", err, closeErr)
		}
		if engine.instance == nil {
			t.Fatalf("instance = nil, want retained instance")
		}
	})

	t.Run("returns unexpected close panic as error", func(t *testing.T) {
		engine := &Engine{
			instance: closeFunc(func() error {
				panic("boom")
			}),
		}

		err := engine.ensureDisabled()
		if err == nil || !strings.Contains(err.Error(), "close engine panic: boom") {
			t.Fatalf("ensureDisabled() error = %v, want close engine panic", err)
		}
		if engine.instance == nil {
			t.Fatalf("instance = nil, want retained instance")
		}
	})
}
