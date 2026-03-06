package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"forge.lthn.ai/core/go-log"
)

// MemoryType represents the classification of a brain memory.
type MemoryType string

const (
	MemoryFact      MemoryType = "fact"
	MemoryDecision  MemoryType = "decision"
	MemoryPattern   MemoryType = "pattern"
	MemoryContext   MemoryType = "context"
	MemoryProcedure MemoryType = "procedure"
)

// Memory represents a single memory entry from the OpenBrain API.
type Memory struct {
	ID           string   `json:"id"`
	AgentID      string   `json:"agent_id,omitempty"`
	Type         string   `json:"type"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	Project      string   `json:"project,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	SupersedesID string   `json:"supersedes_id,omitempty"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

// RememberRequest is the payload for storing a new memory.
type RememberRequest struct {
	Content      string   `json:"content"`
	Type         string   `json:"type"`
	Project      string   `json:"project,omitempty"`
	AgentID      string   `json:"agent_id,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
	SupersedesID string   `json:"supersedes_id,omitempty"`
	Source       string   `json:"source,omitempty"`
}

// RememberResponse is returned after storing a memory.
type RememberResponse struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Project   string `json:"project"`
	CreatedAt string `json:"created_at"`
}

// RecallRequest is the payload for semantic search.
type RecallRequest struct {
	Query         string `json:"query"`
	TopK          int    `json:"top_k,omitempty"`
	Project       string `json:"project,omitempty"`
	Type          string `json:"type,omitempty"`
	AgentID       string `json:"agent_id,omitempty"`
	MinConfidence *float64 `json:"min_confidence,omitempty"`
}

// RecallResponse is returned from a semantic search.
type RecallResponse struct {
	Memories []Memory           `json:"memories"`
	Scores   map[string]float64 `json:"scores"`
}

// Remember stores a memory via POST /v1/brain/remember.
func (c *Client) Remember(ctx context.Context, req RememberRequest) (*RememberResponse, error) {
	const op = "agentic.Client.Remember"

	if req.Content == "" {
		return nil, log.E(op, "content is required", nil)
	}
	if req.Type == "" {
		return nil, log.E(op, "type is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/brain/remember"

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

	var result RememberResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}

// Recall performs semantic search via POST /v1/brain/recall.
func (c *Client) Recall(ctx context.Context, req RecallRequest) (*RecallResponse, error) {
	const op = "agentic.Client.Recall"

	if req.Query == "" {
		return nil, log.E(op, "query is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/brain/recall"

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

	var result RecallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}

// Forget removes a memory via DELETE /v1/brain/forget/{id}.
func (c *Client) Forget(ctx context.Context, id string) error {
	const op = "agentic.Client.Forget"

	if id == "" {
		return log.E(op, "memory ID is required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/brain/forget/%s", c.BaseURL, url.PathEscape(id))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return log.E(op, "failed to create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return log.E(op, "API error", err)
	}

	return nil
}

// EnsureCollection ensures the Qdrant collection exists via POST /v1/brain/collections.
func (c *Client) EnsureCollection(ctx context.Context) error {
	const op = "agentic.Client.EnsureCollection"

	endpoint := c.BaseURL + "/v1/brain/collections"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return log.E(op, "failed to create request", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return log.E(op, "API error", err)
	}

	return nil
}
