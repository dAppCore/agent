package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"forge.lthn.ai/core/go-log"
)

// SessionStatus represents the state of a session.
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionPaused    SessionStatus = "paused"
	SessionCompleted SessionStatus = "completed"
	SessionFailed    SessionStatus = "failed"
)

// Session represents an agent session from the PHP API.
type Session struct {
	SessionID      string        `json:"session_id"`
	AgentType      string        `json:"agent_type"`
	Status         SessionStatus `json:"status"`
	PlanSlug       string        `json:"plan_slug,omitempty"`
	Plan           string        `json:"plan,omitempty"`
	Duration       string        `json:"duration,omitempty"`
	StartedAt      string        `json:"started_at,omitempty"`
	LastActiveAt   string        `json:"last_active_at,omitempty"`
	EndedAt        string        `json:"ended_at,omitempty"`
	ActionCount    int           `json:"action_count,omitempty"`
	ArtifactCount  int           `json:"artifact_count,omitempty"`
	ContextSummary map[string]any `json:"context_summary,omitempty"`
	HandoffNotes   string        `json:"handoff_notes,omitempty"`
	ContinuedFrom  string        `json:"continued_from,omitempty"`
}

// StartSessionRequest is the payload for starting a new session.
type StartSessionRequest struct {
	AgentType string         `json:"agent_type"`
	PlanSlug  string         `json:"plan_slug,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
}

// EndSessionRequest is the payload for ending a session.
type EndSessionRequest struct {
	Status  string `json:"status"`
	Summary string `json:"summary,omitempty"`
}

// ListSessionOptions specifies filters for listing sessions.
type ListSessionOptions struct {
	Status   SessionStatus `json:"status,omitempty"`
	PlanSlug string        `json:"plan_slug,omitempty"`
	Limit    int           `json:"limit,omitempty"`
}

// sessionListResponse wraps the list endpoint response.
type sessionListResponse struct {
	Sessions []Session `json:"sessions"`
	Total    int       `json:"total"`
}

// sessionStartResponse wraps the session create endpoint response.
type sessionStartResponse struct {
	SessionID string `json:"session_id"`
	AgentType string `json:"agent_type"`
	Plan      string `json:"plan,omitempty"`
	Status    string `json:"status"`
}

// sessionEndResponse wraps the session end endpoint response.
type sessionEndResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	Duration  string `json:"duration,omitempty"`
}

// sessionContinueResponse wraps the session continue endpoint response.
type sessionContinueResponse struct {
	SessionID     string `json:"session_id"`
	AgentType     string `json:"agent_type"`
	Plan          string `json:"plan,omitempty"`
	Status        string `json:"status"`
	ContinuedFrom string `json:"continued_from,omitempty"`
}

// ListSessions retrieves sessions matching the given options.
func (c *Client) ListSessions(ctx context.Context, opts ListSessionOptions) ([]Session, error) {
	const op = "agentic.Client.ListSessions"

	params := url.Values{}
	if opts.Status != "" {
		params.Set("status", string(opts.Status))
	}
	if opts.PlanSlug != "" {
		params.Set("plan_slug", opts.PlanSlug)
	}
	if opts.Limit > 0 {
		params.Set("limit", strconv.Itoa(opts.Limit))
	}

	endpoint := c.BaseURL + "/v1/sessions"
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var result sessionListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return result.Sessions, nil
}

// GetSession retrieves a session by ID.
func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	const op = "agentic.Client.GetSession"

	if sessionID == "" {
		return nil, log.E(op, "session ID is required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/sessions/%s", c.BaseURL, url.PathEscape(sessionID))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &session, nil
}

// StartSession starts a new agent session.
func (c *Client) StartSession(ctx context.Context, req StartSessionRequest) (*sessionStartResponse, error) {
	const op = "agentic.Client.StartSession"

	if req.AgentType == "" {
		return nil, log.E(op, "agent_type is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/sessions"

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

	var result sessionStartResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}

// EndSession ends a session with a final status and optional summary.
func (c *Client) EndSession(ctx context.Context, sessionID string, status string, summary string) error {
	const op = "agentic.Client.EndSession"

	if sessionID == "" {
		return log.E(op, "session ID is required", nil)
	}
	if status == "" {
		return log.E(op, "status is required", nil)
	}

	payload := EndSessionRequest{Status: status, Summary: summary}
	data, err := json.Marshal(payload)
	if err != nil {
		return log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/sessions/%s/end", c.BaseURL, url.PathEscape(sessionID))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return log.E(op, "failed to create request", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.checkResponse(resp)
}

// ContinueSession creates a new session continuing from a previous one (multi-agent handoff).
func (c *Client) ContinueSession(ctx context.Context, previousSessionID, agentType string) (*sessionContinueResponse, error) {
	const op = "agentic.Client.ContinueSession"

	if previousSessionID == "" {
		return nil, log.E(op, "previous session ID is required", nil)
	}
	if agentType == "" {
		return nil, log.E(op, "agent_type is required", nil)
	}

	data, err := json.Marshal(map[string]string{"agent_type": agentType})
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/sessions/%s/continue", c.BaseURL, url.PathEscape(previousSessionID))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, log.E(op, "failed to create request", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := c.checkResponse(resp); err != nil {
		return nil, log.E(op, "API error", err)
	}

	var result sessionContinueResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}
