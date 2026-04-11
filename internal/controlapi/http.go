package controlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/infanasotku/farang-edge/internal/engine"
)

type Client struct {
	http      *http.Client
	baseURL   string
	authToken string
}

func New(baseURL, authToken string, httpClient *http.Client) *Client {
	return &Client{
		http:      httpClient,
		baseURL:   baseURL,
		authToken: authToken,
	}
}

func (c *Client) doRequest(ctx context.Context, path string, method string, payload *map[string]any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(b)
	}

	url := c.baseURL + "/api/v1/engines/" + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.authToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
}

func validate(resp *http.Response) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func parse(resp *http.Response, out any) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode json: %w", err)
	}
	return nil
}

type registerInstanceResponse struct {
	Epoch int64 `json:"epoch"`
}

func (c *Client) RegisterInstance(ctx context.Context, engineId uuid.UUID, instanceId uuid.UUID) (int64, error) {
	resp, err := c.doRequest(
		ctx,
		engineId.String()+"/register-instance?instance_id="+instanceId.String(),
		http.MethodPost,
		nil,
	)
	if err != nil {
		return -1, err
	}

	var registerResp registerInstanceResponse
	if err := parse(resp, &registerResp); err != nil {
		return -1, fmt.Errorf("parse response: %w", err)
	}

	return registerResp.Epoch, nil
}

func (c *Client) SendHeartbeat(
	ctx context.Context,
	req engine.HeartbeatRequest,
) error {
	resp, err := c.doRequest(
		ctx,
		req.EngineID.String()+"/heartbeat",
		http.MethodPost,
		&map[string]any{
			"instance_id": req.InstanceID.String(),
			"epoch":       req.Epoch,
			"seq_no":      req.SeqNo,
			"phase":       req.Phase,
			"generation":  req.Generation,
		},
	)
	if err != nil {
		return err
	}

	if err := validate(resp); err != nil {
		return fmt.Errorf("validate response: %w", err)
	}

	return nil
}

type getSpecResponse struct {
	Config     string `json:"config"`
	ConfigHash string `json:"config_hash"`
	Enabled    bool   `json:"enabled"`
	Generation int64  `json:"generation"`
}

func (c *Client) GetSpec(ctx context.Context, engineId uuid.UUID) (engine.SpecSnapshot, error) {
	resp, err := c.doRequest(
		ctx,
		engineId.String()+"/spec",
		http.MethodGet,
		nil,
	)
	var specResp getSpecResponse

	if err != nil {
		return engine.SpecSnapshot{}, err
	}

	if err := parse(resp, &specResp); err != nil {
		return engine.SpecSnapshot{}, fmt.Errorf("parse response: %w", err)
	}

	return engine.SpecSnapshot{
		Config:     specResp.Config,
		ConfigHash: specResp.ConfigHash,
		Enabled:    specResp.Enabled,
		Generation: specResp.Generation,
	}, nil
}
