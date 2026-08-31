package config

import (
	"testing"

	"github.com/google/uuid"
)

func setRequiredEnv(t *testing.T) uuid.UUID {
	t.Helper()

	engineID := uuid.New()
	t.Setenv("ENGINE_ID", engineID.String())
	t.Setenv("CONTROL_BASE_URL", "https://control.example.com")
	t.Setenv("CONTROL_AUTH_TOKEN", "edge-api-key")
	return engineID
}

func TestGetAppConfigReplacementPermit(t *testing.T) {
	t.Run("loads optional permit", func(t *testing.T) {
		engineID := setRequiredEnv(t)
		t.Setenv("REPLACEMENT_PERMIT", "one-time-permit")

		cfg, err := GetAppConfig()
		if err != nil {
			t.Fatalf("GetAppConfig() error = %v", err)
		}

		if cfg.EngineId != engineID {
			t.Fatalf("EngineId = %s, want %s", cfg.EngineId, engineID)
		}
		if cfg.ReplacementPermit != "one-time-permit" {
			t.Fatalf("ReplacementPermit = %q, want one-time-permit", cfg.ReplacementPermit)
		}
	})

	t.Run("allows empty permit", func(t *testing.T) {
		setRequiredEnv(t)
		t.Setenv("REPLACEMENT_PERMIT", "")

		cfg, err := GetAppConfig()
		if err != nil {
			t.Fatalf("GetAppConfig() error = %v", err)
		}

		if cfg.ReplacementPermit != "" {
			t.Fatalf("ReplacementPermit = %q, want empty", cfg.ReplacementPermit)
		}
	})
}
