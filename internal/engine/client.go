package engine

import (
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

func (c *EngineHttpClient) doRequest(ctx context.Context, url string, method string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("X-API-Key", c.authToken)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
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
		c.baseUrl+"/api/v1/engines/"+engineId.String()+"/register-instance?instance_id="+instanceId.String(),
		http.MethodPost,
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
