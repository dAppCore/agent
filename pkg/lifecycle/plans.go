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

// PlanStatus represents the state of a plan.
type PlanStatus string

const (
	PlanDraft     PlanStatus = "draft"
	PlanActive    PlanStatus = "active"
	PlanPaused    PlanStatus = "paused"
	PlanCompleted PlanStatus = "completed"
	PlanArchived  PlanStatus = "archived"
)

// PhaseStatus represents the state of a phase within a plan.
type PhaseStatus string

const (
	PhasePending    PhaseStatus = "pending"
	PhaseInProgress PhaseStatus = "in_progress"
	PhaseCompleted  PhaseStatus = "completed"
	PhaseBlocked    PhaseStatus = "blocked"
	PhaseSkipped    PhaseStatus = "skipped"
)

// Plan represents an agent plan from the PHP API.
type Plan struct {
	Slug         string      `json:"slug"`
	Title        string      `json:"title"`
	Description  string      `json:"description,omitempty"`
	Status       PlanStatus  `json:"status"`
	CurrentPhase int         `json:"current_phase,omitempty"`
	Progress     Progress    `json:"progress,omitempty"`
	Phases       []Phase     `json:"phases,omitempty"`
	Metadata     any         `json:"metadata,omitempty"`
	CreatedAt    string      `json:"created_at,omitempty"`
	UpdatedAt    string      `json:"updated_at,omitempty"`
}

// Phase represents a phase within a plan.
type Phase struct {
	ID                 int           `json:"id,omitempty"`
	Order              int           `json:"order"`
	Name               string        `json:"name"`
	Description        string        `json:"description,omitempty"`
	Status             PhaseStatus   `json:"status"`
	Tasks              []PhaseTask   `json:"tasks,omitempty"`
	TaskProgress       TaskProgress  `json:"task_progress,omitempty"`
	RemainingTasks     []string      `json:"remaining_tasks,omitempty"`
	Dependencies       []int         `json:"dependencies,omitempty"`
	DependencyBlockers []string      `json:"dependency_blockers,omitempty"`
	CanStart           bool          `json:"can_start,omitempty"`
	Checkpoints        []any         `json:"checkpoints,omitempty"`
	StartedAt          string        `json:"started_at,omitempty"`
	CompletedAt        string        `json:"completed_at,omitempty"`
	Metadata           any           `json:"metadata,omitempty"`
}

// PhaseTask represents a task within a phase. Tasks are stored as a JSON array
// in the phase and may be simple strings or objects with status/notes.
type PhaseTask struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
	Notes  string `json:"notes,omitempty"`
}

// UnmarshalJSON handles the fact that tasks can be either strings or objects.
func (t *PhaseTask) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		t.Name = s
		t.Status = "pending"
		return nil
	}

	// Try object
	type taskAlias PhaseTask
	var obj taskAlias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*t = PhaseTask(obj)
	return nil
}

// Progress represents plan progress metrics.
type Progress struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	InProgress int `json:"in_progress"`
	Pending    int `json:"pending"`
	Percentage int `json:"percentage"`
}

// TaskProgress represents task-level progress within a phase.
type TaskProgress struct {
	Total      int `json:"total"`
	Completed  int `json:"completed"`
	Pending    int `json:"pending"`
	Percentage int `json:"percentage"`
}

// ListPlanOptions specifies filters for listing plans.
type ListPlanOptions struct {
	Status          PlanStatus `json:"status,omitempty"`
	IncludeArchived bool       `json:"include_archived,omitempty"`
}

// CreatePlanRequest is the payload for creating a new plan.
type CreatePlanRequest struct {
	Title       string             `json:"title"`
	Slug        string             `json:"slug,omitempty"`
	Description string             `json:"description,omitempty"`
	Context     map[string]any     `json:"context,omitempty"`
	Phases      []CreatePhaseInput `json:"phases,omitempty"`
}

// CreatePhaseInput is a phase definition for plan creation.
type CreatePhaseInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tasks       []string `json:"tasks,omitempty"`
}

// planListResponse wraps the list endpoint response.
type planListResponse struct {
	Plans []Plan `json:"plans"`
	Total int    `json:"total"`
}

// planCreateResponse wraps the create endpoint response.
type planCreateResponse struct {
	Slug   string `json:"slug"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Phases int    `json:"phases"`
}

// planUpdateResponse wraps the update endpoint response.
type planUpdateResponse struct {
	Slug   string `json:"slug"`
	Status string `json:"status"`
}

// planArchiveResponse wraps the archive endpoint response.
type planArchiveResponse struct {
	Slug       string `json:"slug"`
	Status     string `json:"status"`
	ArchivedAt string `json:"archived_at,omitempty"`
}

// ListPlans retrieves plans matching the given options.
func (c *Client) ListPlans(ctx context.Context, opts ListPlanOptions) ([]Plan, error) {
	const op = "agentic.Client.ListPlans"

	params := url.Values{}
	if opts.Status != "" {
		params.Set("status", string(opts.Status))
	}
	if opts.IncludeArchived {
		params.Set("include_archived", "1")
	}

	endpoint := c.BaseURL + "/v1/plans"
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

	var result planListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return result.Plans, nil
}

// GetPlan retrieves a plan by slug (returns full detail with phases).
func (c *Client) GetPlan(ctx context.Context, slug string) (*Plan, error) {
	const op = "agentic.Client.GetPlan"

	if slug == "" {
		return nil, log.E(op, "plan slug is required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s", c.BaseURL, url.PathEscape(slug))

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

	var plan Plan
	if err := json.NewDecoder(resp.Body).Decode(&plan); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &plan, nil
}

// CreatePlan creates a new plan with optional phases.
func (c *Client) CreatePlan(ctx context.Context, req CreatePlanRequest) (*planCreateResponse, error) {
	const op = "agentic.Client.CreatePlan"

	if req.Title == "" {
		return nil, log.E(op, "title is required", nil)
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, log.E(op, "failed to marshal request", err)
	}

	endpoint := c.BaseURL + "/v1/plans"

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

	var result planCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}

// UpdatePlanStatus changes a plan's status.
func (c *Client) UpdatePlanStatus(ctx context.Context, slug string, status PlanStatus) error {
	const op = "agentic.Client.UpdatePlanStatus"

	if slug == "" {
		return log.E(op, "plan slug is required", nil)
	}

	data, err := json.Marshal(map[string]string{"status": string(status)})
	if err != nil {
		return log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s", c.BaseURL, url.PathEscape(slug))

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(data))
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

// ArchivePlan archives a plan with an optional reason.
func (c *Client) ArchivePlan(ctx context.Context, slug string, reason string) error {
	const op = "agentic.Client.ArchivePlan"

	if slug == "" {
		return log.E(op, "plan slug is required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s", c.BaseURL, url.PathEscape(slug))

	var body *bytes.Reader
	if reason != "" {
		data, _ := json.Marshal(map[string]string{"reason": reason})
		body = bytes.NewReader(data)
	}

	var reqBody *bytes.Reader
	if body != nil {
		reqBody = body
	}

	var httpReq *http.Request
	var err error
	if reqBody != nil {
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, reqBody)
		if err != nil {
			return log.E(op, "failed to create request", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
	} else {
		httpReq, err = http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
		if err != nil {
			return log.E(op, "failed to create request", err)
		}
	}
	c.setHeaders(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.checkResponse(resp)
}

// GetPhase retrieves a specific phase within a plan.
func (c *Client) GetPhase(ctx context.Context, planSlug string, phase string) (*Phase, error) {
	const op = "agentic.Client.GetPhase"

	if planSlug == "" || phase == "" {
		return nil, log.E(op, "plan slug and phase identifier are required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s/phases/%s",
		c.BaseURL, url.PathEscape(planSlug), url.PathEscape(phase))

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

	var result Phase
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, log.E(op, "failed to decode response", err)
	}

	return &result, nil
}

// UpdatePhaseStatus changes a phase's status.
func (c *Client) UpdatePhaseStatus(ctx context.Context, planSlug, phase string, status PhaseStatus, notes string) error {
	const op = "agentic.Client.UpdatePhaseStatus"

	if planSlug == "" || phase == "" {
		return log.E(op, "plan slug and phase identifier are required", nil)
	}

	payload := map[string]string{"status": string(status)}
	if notes != "" {
		payload["notes"] = notes
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s/phases/%s",
		c.BaseURL, url.PathEscape(planSlug), url.PathEscape(phase))

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(data))
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

// AddCheckpoint adds a checkpoint note to a phase.
func (c *Client) AddCheckpoint(ctx context.Context, planSlug, phase, note string, checkpointCtx map[string]any) error {
	const op = "agentic.Client.AddCheckpoint"

	if planSlug == "" || phase == "" || note == "" {
		return log.E(op, "plan slug, phase, and note are required", nil)
	}

	payload := map[string]any{"note": note}
	if len(checkpointCtx) > 0 {
		payload["context"] = checkpointCtx
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s/phases/%s/checkpoint",
		c.BaseURL, url.PathEscape(planSlug), url.PathEscape(phase))

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

// UpdateTaskStatus updates a task within a phase.
func (c *Client) UpdateTaskStatus(ctx context.Context, planSlug, phase string, taskIdx int, status string, notes string) error {
	const op = "agentic.Client.UpdateTaskStatus"

	if planSlug == "" || phase == "" {
		return log.E(op, "plan slug and phase are required", nil)
	}

	payload := map[string]any{}
	if status != "" {
		payload["status"] = status
	}
	if notes != "" {
		payload["notes"] = notes
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return log.E(op, "failed to marshal request", err)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s/phases/%s/tasks/%d",
		c.BaseURL, url.PathEscape(planSlug), url.PathEscape(phase), taskIdx)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, endpoint, bytes.NewReader(data))
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

// ToggleTask toggles a task between pending and completed.
func (c *Client) ToggleTask(ctx context.Context, planSlug, phase string, taskIdx int) error {
	const op = "agentic.Client.ToggleTask"

	if planSlug == "" || phase == "" {
		return log.E(op, "plan slug and phase are required", nil)
	}

	endpoint := fmt.Sprintf("%s/v1/plans/%s/phases/%s/tasks/%d/toggle",
		c.BaseURL, url.PathEscape(planSlug), url.PathEscape(phase), taskIdx)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return log.E(op, "failed to create request", err)
	}
	c.setHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return log.E(op, "request failed", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return c.checkResponse(resp)
}
