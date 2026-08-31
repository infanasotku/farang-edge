package controlapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/infanasotku/farang-edge/internal/engine"
)

type recordedRequest struct {
	method            string
	path              string
	instanceID        string
	authToken         string
	replacementPermit string
}

func TestRegisterInstanceReplacementPermit(t *testing.T) {
	tests := []struct {
		name                      string
		configuredPermit          string
		expectedReplacementHeader string
	}{
		{
			name:                      "sends configured permit",
			configuredPermit:          "one-time-permit",
			expectedReplacementHeader: "one-time-permit",
		},
		{
			name:             "omits empty permit",
			configuredPermit: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := make(chan recordedRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- recordedRequest{
					method:            r.Method,
					path:              r.URL.Path,
					instanceID:        r.URL.Query().Get("instance_id"),
					authToken:         r.Header.Get("X-API-Key"),
					replacementPermit: r.Header.Get("X-Replacement-Permit"),
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"epoch":7}`)
			}))
			defer server.Close()

			engineID := uuid.New()
			instanceID := uuid.New()
			client := New(server.URL, "edge-api-key", tt.configuredPermit, server.Client())

			epoch, err := client.RegisterInstance(context.Background(), engineID, instanceID)
			if err != nil {
				t.Fatalf("RegisterInstance() error = %v", err)
			}
			if epoch != 7 {
				t.Fatalf("epoch = %d, want 7", epoch)
			}

			request := <-requests
			if request.method != http.MethodPost {
				t.Fatalf("method = %q, want POST", request.method)
			}
			if request.path != "/api/v1/engines/"+engineID.String()+"/register-instance" {
				t.Fatalf("path = %q", request.path)
			}
			if request.instanceID != instanceID.String() {
				t.Fatalf("instance_id = %q, want %q", request.instanceID, instanceID)
			}
			if request.authToken != "edge-api-key" {
				t.Fatalf("X-API-Key = %q, want edge-api-key", request.authToken)
			}
			if request.replacementPermit != tt.expectedReplacementHeader {
				t.Fatalf(
					"X-Replacement-Permit = %q, want %q",
					request.replacementPermit,
					tt.expectedReplacementHeader,
				)
			}
		})
	}
}

func TestReplacementPermitIsOnlySentDuringRegistration(t *testing.T) {
	permits := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		permits <- r.Header.Get("X-Replacement-Permit")
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = fmt.Fprint(w, `{"config":{},"config_hash":"hash","enabled":false,"generation":1}`)
		}
	}))
	defer server.Close()

	client := New(server.URL, "edge-api-key", "one-time-permit", server.Client())
	engineID := uuid.New()
	if _, err := client.GetSpec(context.Background(), engineID); err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}
	if err := client.SendHeartbeat(context.Background(), engine.HeartbeatRequest{EngineID: engineID}); err != nil {
		t.Fatalf("SendHeartbeat() error = %v", err)
	}

	for range 2 {
		if permit := <-permits; permit != "" {
			t.Fatalf("unexpected X-Replacement-Permit header %q", permit)
		}
	}
}
