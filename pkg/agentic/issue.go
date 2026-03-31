// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// issue := agentic.Issue{Slug: "fix-auth", Title: "Fix auth middleware", Status: "open"}
type Issue struct {
	ID          int            `json:"id"`
	WorkspaceID int            `json:"workspace_id,omitempty"`
	SprintID    int            `json:"sprint_id,omitempty"`
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"`
	Status      string         `json:"status,omitempty"`
	Priority    string         `json:"priority,omitempty"`
	Labels      []string       `json:"labels,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   string         `json:"created_at,omitempty"`
	UpdatedAt   string         `json:"updated_at,omitempty"`
}

// comment := agentic.IssueComment{Author: "codex", Body: "Ready for review"}
type IssueComment struct {
	ID        int            `json:"id"`
	IssueID   int            `json:"issue_id,omitempty"`
	Author    string         `json:"author"`
	Body      string         `json:"body"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
}

// input := agentic.IssueCreateInput{Title: "Fix auth", Type: "bug", Priority: "high"}
type IssueCreateInput struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"`
	Status      string   `json:"status,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	SprintID    int      `json:"sprint_id,omitempty"`
	SprintSlug  string   `json:"sprint_slug,omitempty"`
}

// input := agentic.IssueGetInput{Slug: "fix-auth"}
type IssueGetInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// input := agentic.IssueListInput{Status: "open", Type: "bug"}
type IssueListInput struct {
	Status     string `json:"status,omitempty"`
	Type       string `json:"type,omitempty"`
	SprintID   int    `json:"sprint_id,omitempty"`
	SprintSlug string `json:"sprint_slug,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// input := agentic.IssueUpdateInput{Slug: "fix-auth", Status: "in_progress"}
type IssueUpdateInput struct {
	ID          string   `json:"id,omitempty"`
	Slug        string   `json:"slug,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type,omitempty"`
	Status      string   `json:"status,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	SprintID    int      `json:"sprint_id,omitempty"`
	SprintSlug  string   `json:"sprint_slug,omitempty"`
}

// input := agentic.IssueCommentInput{Slug: "fix-auth", Body: "Ready for review"}
type IssueCommentInput struct {
	ID       string         `json:"id,omitempty"`
	IssueID  string         `json:"issue_id,omitempty"`
	Slug     string         `json:"slug,omitempty"`
	Body     string         `json:"body"`
	Author   string         `json:"author,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// input := agentic.IssueArchiveInput{Slug: "fix-auth"}
type IssueArchiveInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// out := agentic.IssueOutput{Success: true, Issue: agentic.Issue{Slug: "fix-auth"}}
type IssueOutput struct {
	Success bool  `json:"success"`
	Issue   Issue `json:"issue"`
}

// out := agentic.IssueListOutput{Success: true, Count: 1, Issues: []agentic.Issue{{Slug: "fix-auth"}}}
type IssueListOutput struct {
	Success bool    `json:"success"`
	Count   int     `json:"count"`
	Issues  []Issue `json:"issues"`
}

// out := agentic.IssueCommentOutput{Success: true, Comment: agentic.IssueComment{Author: "codex"}}
type IssueCommentOutput struct {
	Success bool         `json:"success"`
	Comment IssueComment `json:"comment"`
}

// out := agentic.IssueArchiveOutput{Success: true, Archived: "fix-auth"}
type IssueArchiveOutput struct {
	Success  bool   `json:"success"`
	Archived string `json:"archived"`
}

// result := c.Action("issue.create").Run(ctx, core.NewOptions(core.Option{Key: "title", Value: "Fix auth"}))
func (s *PrepSubsystem) handleIssueRecordCreate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueCreate(ctx, nil, IssueCreateInput{
		Title:       optionStringValue(options, "title"),
		Description: optionStringValue(options, "description"),
		Type:        optionStringValue(options, "type"),
		Status:      optionStringValue(options, "status"),
		Priority:    optionStringValue(options, "priority"),
		Labels:      optionStringSliceValue(options, "labels"),
		SprintID:    optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug:  optionStringValue(options, "sprint_slug", "sprint-slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("issue.get").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueGet(ctx, nil, IssueGetInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("issue.list").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "open"}))
func (s *PrepSubsystem) handleIssueRecordList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueList(ctx, nil, IssueListInput{
		Status:     optionStringValue(options, "status"),
		Type:       optionStringValue(options, "type"),
		SprintID:   optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug: optionStringValue(options, "sprint_slug", "sprint-slug"),
		Limit:      optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("issue.update").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordUpdate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueUpdate(ctx, nil, IssueUpdateInput{
		ID:          optionStringValue(options, "id", "_arg"),
		Slug:        optionStringValue(options, "slug"),
		Title:       optionStringValue(options, "title"),
		Description: optionStringValue(options, "description"),
		Type:        optionStringValue(options, "type"),
		Status:      optionStringValue(options, "status"),
		Priority:    optionStringValue(options, "priority"),
		Labels:      optionStringSliceValue(options, "labels"),
		SprintID:    optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug:  optionStringValue(options, "sprint_slug", "sprint-slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("issue.comment").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordComment(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueComment(ctx, nil, IssueCommentInput{
		ID:       optionStringValue(options, "id", "_arg"),
		IssueID:  optionStringValue(options, "issue_id", "issue-id"),
		Slug:     optionStringValue(options, "slug"),
		Body:     optionStringValue(options, "body"),
		Author:   optionStringValue(options, "author"),
		Metadata: optionAnyMapValue(options, "metadata"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("issue.archive").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordArchive(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.issueArchive(ctx, nil, IssueArchiveInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerIssueTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_create",
		Description: "Create a tracked platform issue with title, type, priority, labels, and optional sprint assignment.",
	}, s.issueCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_get",
		Description: "Read a tracked platform issue by slug.",
	}, s.issueGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_list",
		Description: "List tracked platform issues with optional status, type, sprint, and limit filters.",
	}, s.issueList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_update",
		Description: "Update fields on a tracked platform issue by slug.",
	}, s.issueUpdate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_comment",
		Description: "Add a comment to a tracked platform issue.",
	}, s.issueComment)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "issue_archive",
		Description: "Archive a tracked platform issue by slug.",
	}, s.issueArchive)
}

func (s *PrepSubsystem) issueCreate(ctx context.Context, _ *mcp.CallToolRequest, input IssueCreateInput) (*mcp.CallToolResult, IssueOutput, error) {
	if input.Title == "" {
		return nil, IssueOutput{}, core.E("issueCreate", "title is required", nil)
	}

	body := map[string]any{
		"title": input.Title,
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if input.Type != "" {
		body["type"] = input.Type
	}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if input.Priority != "" {
		body["priority"] = input.Priority
	}
	if len(input.Labels) > 0 {
		body["labels"] = input.Labels
	}
	if input.SprintID > 0 {
		body["sprint_id"] = input.SprintID
	}
	if input.SprintSlug != "" {
		body["sprint_slug"] = input.SprintSlug
	}

	result := s.platformPayload(ctx, "issue.create", "POST", "/v1/issues", body)
	if !result.OK {
		return nil, IssueOutput{}, resultErrorValue("issue.create", result)
	}

	return nil, IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	}, nil
}

func (s *PrepSubsystem) issueGet(ctx context.Context, _ *mcp.CallToolRequest, input IssueGetInput) (*mcp.CallToolResult, IssueOutput, error) {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, IssueOutput{}, core.E("issueGet", "id or slug is required", nil)
	}

	result := s.platformPayload(ctx, "issue.get", "GET", core.Concat("/v1/issues/", identifier), nil)
	if !result.OK {
		return nil, IssueOutput{}, resultErrorValue("issue.get", result)
	}

	return nil, IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	}, nil
}

func (s *PrepSubsystem) issueList(ctx context.Context, _ *mcp.CallToolRequest, input IssueListInput) (*mcp.CallToolResult, IssueListOutput, error) {
	path := "/v1/issues"
	path = appendQueryParam(path, "status", input.Status)
	path = appendQueryParam(path, "type", input.Type)
	if input.SprintID > 0 {
		path = appendQueryParam(path, "sprint_id", core.Sprint(input.SprintID))
	}
	path = appendQueryParam(path, "sprint_slug", input.SprintSlug)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "issue.list", "GET", path, nil)
	if !result.OK {
		return nil, IssueListOutput{}, resultErrorValue("issue.list", result)
	}

	return nil, parseIssueListOutput(result.Value.(map[string]any)), nil
}

func (s *PrepSubsystem) issueUpdate(ctx context.Context, _ *mcp.CallToolRequest, input IssueUpdateInput) (*mcp.CallToolResult, IssueOutput, error) {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, IssueOutput{}, core.E("issueUpdate", "id or slug is required", nil)
	}

	body := map[string]any{}
	if input.Title != "" {
		body["title"] = input.Title
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if input.Type != "" {
		body["type"] = input.Type
	}
	if input.Status != "" {
		body["status"] = input.Status
	}
	if input.Priority != "" {
		body["priority"] = input.Priority
	}
	if len(input.Labels) > 0 {
		body["labels"] = input.Labels
	}
	if input.SprintID > 0 {
		body["sprint_id"] = input.SprintID
	}
	if input.SprintSlug != "" {
		body["sprint_slug"] = input.SprintSlug
	}
	if len(body) == 0 {
		return nil, IssueOutput{}, core.E("issueUpdate", "at least one field is required", nil)
	}

	result := s.platformPayload(ctx, "issue.update", "PATCH", core.Concat("/v1/issues/", identifier), body)
	if !result.OK {
		return nil, IssueOutput{}, resultErrorValue("issue.update", result)
	}

	return nil, IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	}, nil
}

func (s *PrepSubsystem) issueComment(ctx context.Context, _ *mcp.CallToolRequest, input IssueCommentInput) (*mcp.CallToolResult, IssueCommentOutput, error) {
	identifier := issueRecordIdentifier(input.Slug, input.IssueID, input.ID)
	if identifier == "" {
		return nil, IssueCommentOutput{}, core.E("issueComment", "issue_id, id, or slug is required", nil)
	}
	if input.Body == "" {
		return nil, IssueCommentOutput{}, core.E("issueComment", "body is required", nil)
	}

	body := map[string]any{
		"body": input.Body,
	}
	if input.Author != "" {
		body["author"] = input.Author
	}
	if len(input.Metadata) > 0 {
		body["metadata"] = input.Metadata
	}

	result := s.platformPayload(ctx, "issue.comment", "POST", core.Concat("/v1/issues/", identifier, "/comments"), body)
	if !result.OK {
		return nil, IssueCommentOutput{}, resultErrorValue("issue.comment", result)
	}

	return nil, IssueCommentOutput{
		Success: true,
		Comment: parseIssueComment(payloadResourceMap(result.Value.(map[string]any), "comment")),
	}, nil
}

func (s *PrepSubsystem) issueArchive(ctx context.Context, _ *mcp.CallToolRequest, input IssueArchiveInput) (*mcp.CallToolResult, IssueArchiveOutput, error) {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, IssueArchiveOutput{}, core.E("issueArchive", "id or slug is required", nil)
	}

	result := s.platformPayload(ctx, "issue.archive", "DELETE", core.Concat("/v1/issues/", identifier), nil)
	if !result.OK {
		return nil, IssueArchiveOutput{}, resultErrorValue("issue.archive", result)
	}

	output := IssueArchiveOutput{
		Success:  true,
		Archived: identifier,
	}
	if values := payloadResourceMap(result.Value.(map[string]any), "issue", "result"); len(values) > 0 {
		if slug := stringValue(values["slug"]); slug != "" {
			output.Archived = slug
		}
		if value, ok := boolValueOK(values["success"]); ok {
			output.Success = value
		}
	}
	return nil, output, nil
}

func parseIssue(values map[string]any) Issue {
	return Issue{
		ID:          intValue(values["id"]),
		WorkspaceID: intValue(values["workspace_id"]),
		SprintID:    intValue(values["sprint_id"]),
		Slug:        stringValue(values["slug"]),
		Title:       stringValue(values["title"]),
		Description: stringValue(values["description"]),
		Type:        stringValue(values["type"]),
		Status:      stringValue(values["status"]),
		Priority:    stringValue(values["priority"]),
		Labels:      listValue(values["labels"]),
		Metadata:    anyMapValue(values["metadata"]),
		CreatedAt:   stringValue(values["created_at"]),
		UpdatedAt:   stringValue(values["updated_at"]),
	}
}

func issueRecordIdentifier(values ...string) string {
	for _, value := range values {
		if trimmed := core.Trim(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseIssueComment(values map[string]any) IssueComment {
	return IssueComment{
		ID:        intValue(values["id"]),
		IssueID:   intValue(values["issue_id"]),
		Author:    stringValue(values["author"]),
		Body:      stringValue(values["body"]),
		Metadata:  anyMapValue(values["metadata"]),
		CreatedAt: stringValue(values["created_at"]),
	}
}

func parseIssueListOutput(payload map[string]any) IssueListOutput {
	issuesData := payloadDataSlice(payload, "issues")
	issues := make([]Issue, 0, len(issuesData))
	for _, values := range issuesData {
		issues = append(issues, parseIssue(values))
	}

	count := mapIntValue(payload, "total", "count")
	if count == 0 {
		count = mapIntValue(payloadDataMap(payload), "total", "count")
	}
	if count == 0 {
		count = len(issues)
	}

	return IssueListOutput{
		Success: true,
		Count:   count,
		Issues:  issues,
	}
}
