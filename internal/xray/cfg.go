package xray

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/infanasotku/farang-edge/internal/engine"
)

type Builder struct {
	Runtime RuntimeConfig
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
		"listen":   b.Runtime.APIListen,
		"services": []string{"ObservatoryService"},
	}

	doc["observatory"] = map[string]any{
		"subjectSelector":   b.Runtime.SubjectSelectors,
		"probeURL":          b.Runtime.ProbeURL,
		"probeInterval":     b.Runtime.ProbeInterval.String(),
		"enableConcurrency": b.Runtime.EnableConcurrency,
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return engine.BuildResult{}, fmt.Errorf("marshal effective xray config: %w", err)
	}

	effectiveHash := newSHA256([]byte(
		"remote=" + remoteHash + "\n" +
			"overlay_version=v1\n" +
			"api_listen=" + b.Runtime.APIListen + "\n" +
			"probe_url=" + b.Runtime.ProbeURL + "\n" +
			"probe_interval=" + b.Runtime.ProbeInterval.String() + "\n" +
			"selectors=" + strings.Join(b.Runtime.SubjectSelectors, ",") + "\n" +
			fmt.Sprintf("enable_concurrency=%t", b.Runtime.EnableConcurrency),
	))

	return engine.BuildResult{
		Config: string(raw),
		Hash:   effectiveHash,
	}, nil
}
