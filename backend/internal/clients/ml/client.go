package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config defines settings for connecting to the ML optimization service.
type Config struct {
	BaseURL string
	Timeout time.Duration
}

// Client implements MLClient using an HTTP transport.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new ML service HTTP client with timeout limited to at most 5 seconds.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	timeout := cfg.Timeout
	if timeout <= 0 || timeout > 5*time.Second {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// OptimizeRoute sends candidates and user profiles to the ML service to build an optimized POI sequence.
func (c *Client) OptimizeRoute(ctx context.Context, req *OptimizeRequest) (*OptimizeResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("optimize request cannot be nil")
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal optimize request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/optimize", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ml service request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ml response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml service returned non-200 status %d: %s", resp.StatusCode, string(respBytes))
	}

	var optimizeResp OptimizeResponse
	if err := json.Unmarshal(respBytes, &optimizeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ml response: %w", err)
	}

	return &optimizeResp, nil
}
