package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type EngineHttpClient struct {
	http      *http.Client
	baseUrl   string
	authToken string
}

func NewClient(baseUrl string, authToken string) *EngineHttpClient {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
	}

	return &EngineHttpClient{
		baseUrl:   baseUrl,
		authToken: authToken,
		http:      &http.Client{Transport: transport},
	}
}

func (c *EngineHttpClient) doRequest(ctx context.Context, path string, method string, payload *map[string]any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(b)
	}

	url := c.baseUrl + "/api/v1/engines/" + path
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

func (c *EngineHttpClient) RegisterInstance(ctx context.Context, engineId uuid.UUID, instanceId uuid.UUID) (int64, error) {
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

func (c *EngineHttpClient) SendHeartbeat(
	ctx context.Context,
	engineId uuid.UUID,
	instanceId uuid.UUID,
	epoch int64,
	seq_no int64,
	phase string,
	generation int64,
) error {
	resp, err := c.doRequest(
		ctx,
		engineId.String()+"/heartbeat",
		http.MethodPost,
		&map[string]any{
			"instance_id": instanceId.String(),
			"epoch":       epoch,
			"seq_no":      seq_no,
			"phase":       phase,
			"generation":  generation,
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
	Config     map[string]interface{} `json:"config"`
	Enabled    bool                   `json:"enabled"`
	Generation int64                  `json:"generation"`
	ConfigHash string                 `json:"config_hash"`
}

func (c *EngineHttpClient) GetSpec(ctx context.Context, engineId uuid.UUID) (*getSpecResponse, error) {
	resp, err := c.doRequest(
		ctx,
		engineId.String()+"/spec",
		http.MethodGet,
		nil,
	)
	if err != nil {
		return nil, err
	}

	var specResp getSpecResponse
	if err := parse(resp, &specResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &specResp, nil
}
