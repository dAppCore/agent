package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"forge.lthn.ai/core/go-log"
)

// ScoreContentRequest is the payload for content scoring.
type ScoreContentRequest struct {
	Text   string `json:"text"`
	Prompt string `json:"prompt,omitempty"`
}

// ScoreImprintRequest is the payload for linguistic imprint analysis.
type ScoreImprintRequest struct {
	Text string `json:"text"`
}

// ScoreResult holds the response from the scoring engine.
// The shape is proxied from the EaaS Go binary, so fields are dynamic.
type ScoreResult map[string]any

// ScoreHealthResponse holds the health check result.
type ScoreHealthResponse struct {
	Status         string `json:"status"`
	UpstreamStatus int    `json:"upstream_status,omitempty"`
}

// ScoreContent scores text for AI patterns via POST /v1/score/content.
func (c *Client) ScoreContent(ctx context.Context, req ScoreContentRequest) (ScoreResult, error) {
	const op = "agentic.Client.ScoreContent"

	if req.Text == "" {
		return nil, log.E(op, "text is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/score/content"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var result ScoreResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return result, nil
}

// ScoreImprint performs linguistic imprint analysis via POST /v1/score/imprint.
func (c *Client) ScoreImprint(ctx context.Context, req ScoreImprintRequest) (ScoreResult, error) {
	const op = "agentic.Client.ScoreImprint"

	if req.Text == "" {
		return nil, log.E(op, "text is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/score/imprint"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}

	c.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var result ScoreResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return result, nil
}

// ScoreHealth checks the scoring engine health via GET /v1/score/health.
// This endpoint does not require authentication.
func (c *Client) ScoreHealth(ctx context.Context) (*ScoreHealthResponse, error) {
	const op = "agentic.Client.ScoreHealth"

	endpoint := c.BaseURL + "/v1/score/health"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}

	// Health endpoint is unauthenticated but we still set headers for consistency.
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "core-agentic-client/1.0")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var result ScoreHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}
