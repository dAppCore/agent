// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sprint := agentic.Sprint{Slug: "ax-follow-up", Title: "AX Follow-up", Status: "active"}
type Sprint struct {
	ID          int            `json:"id"`
	WorkspaceID int            `json:"workspace_id,omitempty"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Goal        string         `json:"goal,omitempty"`
	Status      string         `json:"status,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	StartedAt   string         `json:"started_at,omitempty"`
	EndedAt     string         `json:"ended_at,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
}

// input := agentic.SprintCreateInput{Title: "AX Follow-up", Goal: "Finish RFC parity"}
type SprintCreateInput struct {
	Title     string         `json:"title"`
	Goal      string         `json:"goal,omitempty"`
	Status    string         `json:"status,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	StartedAt string         `json:"started_at,omitempty"`
	EndedAt   string         `json:"ended_at,omitempty"`
}

// input := agentic.SprintGetInput{Slug: "ax-follow-up"}
type SprintGetInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// input := agentic.SprintListInput{Status: "active", Limit: 10}
type SprintListInput struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// input := agentic.SprintUpdateInput{Slug: "ax-follow-up", Status: "completed"}
type SprintUpdateInput struct {
	ID        string         `json:"id,omitempty"`
	Slug      string         `json:"slug,omitempty"`
	Title     string         `json:"title,omitempty"`
	Goal      string         `json:"goal,omitempty"`
	Status    string         `json:"status,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	StartedAt string         `json:"started_at,omitempty"`
	EndedAt   string         `json:"ended_at,omitempty"`
}

// input := agentic.SprintArchiveInput{Slug: "ax-follow-up"}
type SprintArchiveInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// out := agentic.SprintOutput{Success: true, Sprint: agentic.Sprint{Slug: "ax-follow-up"}}
type SprintOutput struct {
	Success bool   `json:"success"`
	Sprint  Sprint `json:"sprint"`
}

// out := agentic.SprintListOutput{Success: true, Count: 1, Sprints: []agentic.Sprint{{Slug: "ax-follow-up"}}}
type SprintListOutput struct {
	Success bool     `json:"success"`
	Count   int      `json:"count"`
	Sprints []Sprint `json:"sprints"`
}

// out := agentic.SprintArchiveOutput{Success: true, Archived: "ax-follow-up"}
type SprintArchiveOutput struct {
	Success  bool   `json:"success"`
	Archived string `json:"archived"`
}

// result := c.Action("sprint.create").Run(ctx, core.NewOptions(core.Option{Key: "title", Value: "AX Follow-up"}))
func (s *PrepSubsystem) handleSprintCreate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sprintCreate(ctx, nil, SprintCreateInput{
		Title:     optionStringValue(options, "title"),
		Goal:      optionStringValue(options, "goal"),
		Status:    optionStringValue(options, "status"),
		Metadata:  optionAnyMapValue(options, "metadata"),
		StartedAt: optionStringValue(options, "started_at", "started-at"),
		EndedAt:   optionStringValue(options, "ended_at", "ended-at"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("sprint.get").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "ax-follow-up"}))
func (s *PrepSubsystem) handleSprintGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sprintGet(ctx, nil, SprintGetInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("sprint.list").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "active"}))
func (s *PrepSubsystem) handleSprintList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sprintList(ctx, nil, SprintListInput{
		Status: optionStringValue(options, "status"),
		Limit:  optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("sprint.update").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "ax-follow-up"}))
func (s *PrepSubsystem) handleSprintUpdate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sprintUpdate(ctx, nil, SprintUpdateInput{
		ID:        optionStringValue(options, "id", "_arg"),
		Slug:      optionStringValue(options, "slug"),
		Title:     optionStringValue(options, "title"),
		Goal:      optionStringValue(options, "goal"),
		Status:    optionStringValue(options, "status"),
		Metadata:  optionAnyMapValue(options, "metadata"),
		StartedAt: optionStringValue(options, "started_at", "started-at"),
		EndedAt:   optionStringValue(options, "ended_at", "ended-at"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("sprint.archive").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "ax-follow-up"}))
func (s *PrepSubsystem) handleSprintArchive(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.sprintArchive(ctx, nil, SprintArchiveInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerSprintTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "sprint_create",
		Description: "Create a tracked platform sprint with goal, schedule, and metadata.",
	}, s.sprintCreate)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_sprint_create",
		Description: "Create a tracked platform sprint with goal, schedule, and metadata.",
	}, s.sprintCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sprint_get",
		Description: "Read a tracked platform sprint by slug.",
	}, s.sprintGet)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_sprint_get",
		Description: "Read a tracked platform sprint by slug.",
	}, s.sprintGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sprint_list",
		Description: "List tracked platform sprints with optional status and limit filters.",
	}, s.sprintList)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_sprint_list",
		Description: "List tracked platform sprints with optional status and limit filters.",
	}, s.sprintList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sprint_update",
		Description: "Update fields on a tracked platform sprint by slug.",
	}, s.sprintUpdate)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_sprint_update",
		Description: "Update fields on a tracked platform sprint by slug.",
	}, s.sprintUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "sprint_archive",
		Description: "Archive a tracked platform sprint by slug.",
	}, s.sprintArchive)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_sprint_archive",
		Description: "Archive a tracked platform sprint by slug.",
	}, s.sprintArchive)
}

func (s *PrepSubsystem) sprintCreate(ctx context.Context, _ *mcp.CallToolRequest, input SprintCreateInput) (*mcp.CallToolResult, SprintOutput, error) {
	if input.Title == "" {
		return nil, SprintOutput{}, core.E("sprintCreate", "title is required", nil)
	}

	body := map[string]any{
		"title": input.Title,
	}
	if input.Goal != "" {
		body["goal"] = input.Goal
	}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if len(input.Metadata) > 0 {
		body["metadata"] = input.Metadata
	}
	if input.StartedAt != "" {
		body["started_at"] = input.StartedAt
	}
	if input.EndedAt != "" {
		body["ended_at"] = input.EndedAt
	}

	result := s.platformPayload(ctx, "sprint.create", "POST", "/v1/sprints", body)
	if !result.OK {
		return nil, SprintOutput{}, resultErrorValue("sprint.create", result)
	}

	return nil, SprintOutput{
		Success: true,
		Sprint:  parseSprint(payloadResourceMap(result.Value.(map[string]any), "sprint")),
	}, nil
}

func (s *PrepSubsystem) sprintGet(ctx context.Context, _ *mcp.CallToolRequest, input SprintGetInput) (*mcp.CallToolResult, SprintOutput, error) {
	identifier := sprintIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, SprintOutput{}, core.E("sprintGet", "id or slug is required", nil)
	}

	result := s.platformPayload(ctx, "sprint.get", "GET", core.Concat("/v1/sprints/", identifier), nil)
	if !result.OK {
		return nil, SprintOutput{}, resultErrorValue("sprint.get", result)
	}

	return nil, SprintOutput{
		Success: true,
		Sprint:  parseSprint(payloadResourceMap(result.Value.(map[string]any), "sprint")),
	}, nil
}

func (s *PrepSubsystem) sprintList(ctx context.Context, _ *mcp.CallToolRequest, input SprintListInput) (*mcp.CallToolResult, SprintListOutput, error) {
	path := "/v1/sprints"
	path = appendQueryParam(path, "status", input.Status)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "sprint.list", "GET", path, nil)
	if !result.OK {
		return nil, SprintListOutput{}, resultErrorValue("sprint.list", result)
	}

	return nil, parseSprintListOutput(result.Value.(map[string]any)), nil
}

func (s *PrepSubsystem) sprintUpdate(ctx context.Context, _ *mcp.CallToolRequest, input SprintUpdateInput) (*mcp.CallToolResult, SprintOutput, error) {
	identifier := sprintIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, SprintOutput{}, core.E("sprintUpdate", "id or slug is required", nil)
	}

	body := map[string]any{}
	if input.Title != "" {
		body["title"] = input.Title
	}
	if input.Goal != "" {
		body["goal"] = input.Goal
	}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if len(input.Metadata) > 0 {
		body["metadata"] = input.Metadata
	}
	if input.StartedAt != "" {
		body["started_at"] = input.StartedAt
	}
	if input.EndedAt != "" {
		body["ended_at"] = input.EndedAt
	}
	if len(body) == 0 {
		return nil, SprintOutput{}, core.E("sprintUpdate", "at least one field is required", nil)
	}

	result := s.platformPayload(ctx, "sprint.update", "PATCH", core.Concat("/v1/sprints/", identifier), body)
	if !result.OK {
		return nil, SprintOutput{}, resultErrorValue("sprint.update", result)
	}

	return nil, SprintOutput{
		Success: true,
		Sprint:  parseSprint(payloadResourceMap(result.Value.(map[string]any), "sprint")),
	}, nil
}

func (s *PrepSubsystem) sprintArchive(ctx context.Context, _ *mcp.CallToolRequest, input SprintArchiveInput) (*mcp.CallToolResult, SprintArchiveOutput, error) {
	identifier := sprintIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, SprintArchiveOutput{}, core.E("sprintArchive", "id or slug is required", nil)
	}

	result := s.platformPayload(ctx, "sprint.archive", "DELETE", core.Concat("/v1/sprints/", identifier), nil)
	if !result.OK {
		return nil, SprintArchiveOutput{}, resultErrorValue("sprint.archive", result)
	}

	output := SprintArchiveOutput{
		Success:  true,
		Archived: identifier,
	}
	if values := payloadResourceMap(result.Value.(map[string]any), "sprint", "result"); len(values) > 0 {
		if slug := stringValue(values["slug"]); slug != "" {
			output.Archived = slug
		}
		if value, ok := boolValueOK(values["success"]); ok {
			output.Success = value
		}
	}
	return nil, output, nil
}

func sprintIdentifier(values ...string) string {
	for _, value := range values {
		if trimmed := core.Trim(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseSprint(values map[string]any) Sprint {
	return Sprint{
		ID:          intValue(values["id"]),
		WorkspaceID: intValue(values["workspace_id"]),
		Slug:        stringValue(values["slug"]),
		Title:       stringValue(values["title"]),
		Goal:        stringValue(values["goal"]),
		Status:      stringValue(values["status"]),
		Metadata:    anyMapValue(values["metadata"]),
		StartedAt:   stringValue(values["started_at"]),
		EndedAt:     stringValue(values["ended_at"]),
		CreatedAt:   stringValue(values["created_at"]),
		UpdatedAt:   stringValue(values["updated_at"]),
	}
}

func parseSprintListOutput(payload map[string]any) SprintListOutput {
	sprintsData := payloadDataSlice(payload, "sprints")
	sprints := make([]Sprint, 0, len(sprintsData))
	for _, values := range sprintsData {
		sprints = append(sprints, parseSprint(values))
	}

	count := mapIntValue(payload, "total", "count")
	if count == 0 {
		count = mapIntValue(payloadDataMap(payload), "total", "count")
	}
	if count == 0 {
		count = len(sprints)
	}

	return SprintListOutput{
		Success: true,
		Count:   count,
		Sprints: sprints,
	}
}
