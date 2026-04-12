package xray

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/infanasotku/farang-edge/internal/engine"
)

type OverlayConfig struct {
	APIListen         string
	ProbeURL          string
	ProbeInterval     time.Duration
	SubjectSelectors  []string
	EnableConcurrency bool
}

type Builder struct {
	Overlay OverlayConfig
}

func newSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return string(hash[:])
}

func (b *Builder) Build(remoteConfig string, remoteHash string) (engine.BuildResult, error) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(remoteConfig), &doc); err != nil {
		return engine.BuildResult{}, fmt.Errorf("parse remote xray config: %w", err)
	}

	for _, k := range []string{"api", "observatory"} {
		if _, exists := doc[k]; exists {
			return engine.BuildResult{}, fmt.Errorf("remote config must not define reserved top-level key %q", k)
		}
	}

	doc["api"] = map[string]any{
		"tag":      "api",
		"listen":   b.Overlay.APIListen,
		"services": []string{"ObservatoryService"},
	}

	doc["observatory"] = map[string]any{
		"subjectSelector":   b.Overlay.SubjectSelectors,
		"probeURL":          b.Overlay.ProbeURL,
		"probeInterval":     b.Overlay.ProbeInterval.String(),
		"enableConcurrency": b.Overlay.EnableConcurrency,
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return engine.BuildResult{}, fmt.Errorf("marshal effective xray config: %w", err)
	}

	effectiveHash := newSHA256([]byte(
		"remote=" + remoteHash + "\n" +
			"overlay_version=v1\n" +
			"api_listen=" + b.Overlay.APIListen + "\n" +
			"probe_url=" + b.Overlay.ProbeURL + "\n" +
			"probe_interval=" + b.Overlay.ProbeInterval.String() + "\n" +
			"selectors=" + strings.Join(b.Overlay.SubjectSelectors, ",") + "\n" +
			fmt.Sprintf("enable_concurrency=%t", b.Overlay.EnableConcurrency),
	))

	return engine.BuildResult{
		Config: string(raw),
		Hash:   effectiveHash,
	}, nil
}
