// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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
	Assignee    string         `json:"assignee,omitempty"`
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
	Assignee    string   `json:"assignee,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	SprintID    int      `json:"sprint_id,omitempty"`
	SprintSlug  string   `json:"sprint_slug,omitempty"`
}

// input := agentic.IssueGetInput{Slug: "fix-auth"}
type IssueGetInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// input := agentic.IssueListInput{Status: "open", Type: "bug", Priority: "high", Labels: []string{"auth"}}
type IssueListInput struct {
	Status     string   `json:"status,omitempty"`
	Type       string   `json:"type,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	Assignee   string   `json:"assignee,omitempty"`
	Labels     []string `json:"labels,omitempty"`
	SprintID   int      `json:"sprint_id,omitempty"`
	SprintSlug string   `json:"sprint_slug,omitempty"`
	Limit      int      `json:"limit,omitempty"`
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
	Assignee    string   `json:"assignee,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	SprintID    int      `json:"sprint_id,omitempty"`
	SprintSlug  string   `json:"sprint_slug,omitempty"`
}

// input := agentic.IssueAssignInput{Slug: "fix-auth", Assignee: "codex"}
type IssueAssignInput struct {
	ID       string `json:"id,omitempty"`
	Slug     string `json:"slug,omitempty"`
	Assignee string `json:"assignee,omitempty"`
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

// input := agentic.IssueReportInput{Slug: "fix-auth", Report: map[string]any{"summary": "Build failed"}}
type IssueReportInput struct {
	ID       string         `json:"id,omitempty"`
	IssueID  string         `json:"issue_id,omitempty"`
	Slug     string         `json:"slug,omitempty"`
	Report   any            `json:"report,omitempty"`
	Author   string         `json:"author,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// out := agentic.IssueReportOutput{Success: true, Comment: agentic.IssueComment{Author: "codex"}}
type IssueReportOutput struct {
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
	return typedResultValue[IssueOutput]("issue.create", "invalid issue create output", s.issueCreate(ctx, IssueCreateInput{
		Title:       optionStringValue(options, "title"),
		Description: optionStringValue(options, "description"),
		Type:        optionStringValue(options, "type"),
		Status:      optionStringValue(options, "status"),
		Priority:    optionStringValue(options, "priority"),
		Assignee:    optionStringValue(options, "assignee"),
		Labels:      optionStringSliceValue(options, "labels"),
		SprintID:    optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug:  optionStringValue(options, "sprint_slug", "sprint-slug"),
	}))
}

// result := c.Action("issue.get").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordGet(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueOutput]("issue.get", "invalid issue get output", s.issueGet(ctx, IssueGetInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	}))
}

// result := c.Action("issue.list").Run(ctx, core.NewOptions(core.Option{Key: "status", Value: "open"}))
func (s *PrepSubsystem) handleIssueRecordList(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueListOutput]("issue.list", "invalid issue list output", s.issueList(ctx, IssueListInput{
		Status:     optionStringValue(options, "status"),
		Type:       optionStringValue(options, "type"),
		Priority:   optionStringValue(options, "priority"),
		Assignee:   optionStringValue(options, "assignee", "agent", "agent_type"),
		Labels:     optionStringSliceValue(options, "labels"),
		SprintID:   optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug: optionStringValue(options, "sprint_slug", "sprint-slug"),
		Limit:      optionIntValue(options, "limit"),
	}))
}

// result := c.Action("issue.update").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordUpdate(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueOutput]("issue.update", "invalid issue update output", s.issueUpdate(ctx, IssueUpdateInput{
		ID:          optionStringValue(options, "id", "_arg"),
		Slug:        optionStringValue(options, "slug"),
		Title:       optionStringValue(options, "title"),
		Description: optionStringValue(options, "description"),
		Type:        optionStringValue(options, "type"),
		Status:      optionStringValue(options, "status"),
		Priority:    optionStringValue(options, "priority"),
		Assignee:    optionStringValue(options, "assignee"),
		Labels:      optionStringSliceValue(options, "labels"),
		SprintID:    optionIntValue(options, "sprint_id", "sprint-id"),
		SprintSlug:  optionStringValue(options, "sprint_slug", "sprint-slug"),
	}))
}

// result := c.Action("issue.assign").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "slug", Value: "fix-auth"},
//	core.Option{Key: "assignee", Value: "codex"},
//
// ))
func (s *PrepSubsystem) handleIssueRecordAssign(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueOutput]("issue.assign", "invalid issue assign output", s.issueAssign(ctx, IssueAssignInput{
		ID:       optionStringValue(options, "id", "_arg"),
		Slug:     optionStringValue(options, "slug"),
		Assignee: optionStringValue(options, "assignee", "agent", "agent_type"),
	}))
}

// result := c.Action("issue.comment").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordComment(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueCommentOutput]("issue.comment", "invalid issue comment output", s.issueComment(ctx, IssueCommentInput{
		ID:       optionStringValue(options, "id", "_arg"),
		IssueID:  optionStringValue(options, "issue_id", "issue-id"),
		Slug:     optionStringValue(options, "slug"),
		Body:     optionStringValue(options, "body"),
		Author:   optionStringValue(options, "author"),
		Metadata: optionAnyMapValue(options, "metadata"),
	}))
}

// result := c.Action("issue.report").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "slug", Value: "fix-auth"},
//	core.Option{Key: "report", Value: map[string]any{"summary": "Build failed"}},
//
// ))
func (s *PrepSubsystem) handleIssueRecordReport(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueReportOutput]("issue.report", "invalid issue report output", s.issueReport(ctx, IssueReportInput{
		ID:       optionStringValue(options, "id", "_arg"),
		IssueID:  optionStringValue(options, "issue_id", "issue-id"),
		Slug:     optionStringValue(options, "slug"),
		Report:   optionAnyValue(options, "report", "body"),
		Author:   optionStringValue(options, "author"),
		Metadata: optionAnyMapValue(options, "metadata"),
	}))
}

// result := c.Action("issue.archive").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handleIssueRecordArchive(ctx context.Context, options core.Options) core.Result {
	return typedResultValue[IssueArchiveOutput]("issue.archive", "invalid issue archive output", s.issueArchive(ctx, IssueArchiveInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	}))
}

func (s *PrepSubsystem) registerIssueTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_create",
		Description: "Create a tracked platform issue with title, type, priority, labels, and optional sprint assignment.",
	}, toolHandlerFor[IssueCreateInput, IssueOutput]("issue.create", "invalid issue create output", s.issueCreate))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_create",
		Description: "Create a tracked platform issue with title, type, priority, labels, and optional sprint assignment.",
	}, toolHandlerFor[IssueCreateInput, IssueOutput]("issue.create", "invalid issue create output", s.issueCreate))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_get",
		Description: "Read a tracked platform issue by slug.",
	}, toolHandlerFor[IssueGetInput, IssueOutput]("issue.get", "invalid issue get output", s.issueGet))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_get",
		Description: "Read a tracked platform issue by slug.",
	}, toolHandlerFor[IssueGetInput, IssueOutput]("issue.get", "invalid issue get output", s.issueGet))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_list",
		Description: "List tracked platform issues with optional status, type, sprint, and limit filters.",
	}, toolHandlerFor[IssueListInput, IssueListOutput]("issue.list", "invalid issue list output", s.issueList))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_list",
		Description: "List tracked platform issues with optional status, type, sprint, and limit filters.",
	}, toolHandlerFor[IssueListInput, IssueListOutput]("issue.list", "invalid issue list output", s.issueList))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_update",
		Description: "Update fields on a tracked platform issue by slug.",
	}, toolHandlerFor[IssueUpdateInput, IssueOutput]("issue.update", "invalid issue update output", s.issueUpdate))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_update",
		Description: "Update fields on a tracked platform issue by slug.",
	}, toolHandlerFor[IssueUpdateInput, IssueOutput]("issue.update", "invalid issue update output", s.issueUpdate))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_assign",
		Description: "Assign an agent or user to a tracked platform issue by slug.",
	}, toolHandlerFor[IssueAssignInput, IssueOutput]("issue.assign", "invalid issue assign output", s.issueAssign))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_assign",
		Description: "Assign an agent or user to a tracked platform issue by slug.",
	}, toolHandlerFor[IssueAssignInput, IssueOutput]("issue.assign", "invalid issue assign output", s.issueAssign))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_comment",
		Description: "Add a comment to a tracked platform issue.",
	}, toolHandlerFor[IssueCommentInput, IssueCommentOutput]("issue.comment", "invalid issue comment output", s.issueComment))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_comment",
		Description: "Add a comment to a tracked platform issue.",
	}, toolHandlerFor[IssueCommentInput, IssueCommentOutput]("issue.comment", "invalid issue comment output", s.issueComment))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_report",
		Description: "Post a structured report comment to a tracked platform issue.",
	}, toolHandlerFor[IssueReportInput, IssueReportOutput]("issue.report", "invalid issue report output", s.issueReport))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_report",
		Description: "Post a structured report comment to a tracked platform issue.",
	}, toolHandlerFor[IssueReportInput, IssueReportOutput]("issue.report", "invalid issue report output", s.issueReport))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "issue_archive",
		Description: "Archive a tracked platform issue by slug.",
	}, toolHandlerFor[IssueArchiveInput, IssueArchiveOutput]("issue.archive", "invalid issue archive output", s.issueArchive))
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_issue_archive",
		Description: "Archive a tracked platform issue by slug.",
	}, toolHandlerFor[IssueArchiveInput, IssueArchiveOutput]("issue.archive", "invalid issue archive output", s.issueArchive))
}

func (s *PrepSubsystem) issueCreate(ctx context.Context, input IssueCreateInput) core.Result {
	if input.Title == "" {
		return core.Fail(core.E("issueCreate", "title is required", nil))
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
	if input.Assignee != "" {
		body["assignee"] = input.Assignee
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
		return failureResult("issue.create", "request failed", result)
	}

	return core.Ok(IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	})
}

func (s *PrepSubsystem) issueGet(ctx context.Context, input IssueGetInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueGet", "id or slug is required", nil))
	}

	result := s.platformPayload(ctx, "issue.get", "GET", core.Concat("/v1/issues/", identifier), nil)
	if !result.OK {
		return failureResult("issue.get", "request failed", result)
	}

	return core.Ok(IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	})
}

func (s *PrepSubsystem) issueList(ctx context.Context, input IssueListInput) core.Result {
	path := "/v1/issues"
	path = appendQueryParam(path, "status", input.Status)
	path = appendQueryParam(path, "type", input.Type)
	path = appendQueryParam(path, "priority", input.Priority)
	path = appendQueryParam(path, "assignee", input.Assignee)
	path = appendQuerySlice(path, "labels", input.Labels)
	if input.SprintID > 0 {
		path = appendQueryParam(path, "sprint_id", core.Sprint(input.SprintID))
	}
	path = appendQueryParam(path, "sprint_slug", input.SprintSlug)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "issue.list", "GET", path, nil)
	if !result.OK {
		return failureResult("issue.list", "request failed", result)
	}

	return core.Ok(parseIssueListOutput(result.Value.(map[string]any)))
}

func (s *PrepSubsystem) issueUpdate(ctx context.Context, input IssueUpdateInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueUpdate", "id or slug is required", nil))
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
	if input.Assignee != "" {
		body["assignee"] = input.Assignee
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
		return core.Fail(core.E("issueUpdate", "at least one field is required", nil))
	}

	result := s.platformPayload(ctx, "issue.update", "PATCH", core.Concat("/v1/issues/", identifier), body)
	if !result.OK {
		return failureResult("issue.update", "request failed", result)
	}

	return core.Ok(IssueOutput{
		Success: true,
		Issue:   parseIssue(payloadResourceMap(result.Value.(map[string]any), "issue")),
	})
}

func (s *PrepSubsystem) issueAssign(ctx context.Context, input IssueAssignInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueAssign", "id or slug is required", nil))
	}
	if input.Assignee == "" {
		return core.Fail(core.E("issueAssign", "assignee is required", nil))
	}

	return s.issueUpdate(ctx, IssueUpdateInput{
		ID:       input.ID,
		Slug:     input.Slug,
		Assignee: input.Assignee,
	})
}

func (s *PrepSubsystem) issueComment(ctx context.Context, input IssueCommentInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.IssueID, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueComment", "issue_id, id, or slug is required", nil))
	}
	if input.Body == "" {
		return core.Fail(core.E("issueComment", "body is required", nil))
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
		return failureResult("issue.comment", "request failed", result)
	}

	return core.Ok(IssueCommentOutput{
		Success: true,
		Comment: parseIssueComment(payloadResourceMap(result.Value.(map[string]any), "comment")),
	})
}

func (s *PrepSubsystem) issueReport(ctx context.Context, input IssueReportInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.IssueID, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueReport", "issue_id, id, or slug is required", nil))
	}

	bodyResult := issueReportBody(input.Report)
	if !bodyResult.OK {
		return bodyResult
	}
	body, _ := bodyResult.Value.(string)
	if body == "" {
		return core.Fail(core.E("issueReport", "report is required", nil))
	}

	commentResult := typedResultValue[IssueCommentOutput]("issue.report", "invalid issue comment output", s.issueComment(ctx, IssueCommentInput{
		ID:       input.ID,
		IssueID:  input.IssueID,
		Slug:     input.Slug,
		Body:     body,
		Author:   input.Author,
		Metadata: input.Metadata,
	}))
	if !commentResult.OK {
		return commentResult
	}
	commentOutput, _ := commentResult.Value.(IssueCommentOutput)

	return core.Ok(IssueReportOutput{
		Success: true,
		Comment: commentOutput.Comment,
	})
}

func (s *PrepSubsystem) issueArchive(ctx context.Context, input IssueArchiveInput) core.Result {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return core.Fail(core.E("issueArchive", "id or slug is required", nil))
	}

	result := s.platformPayload(ctx, "issue.archive", "DELETE", core.Concat("/v1/issues/", identifier), nil)
	if !result.OK {
		return failureResult("issue.archive", "request failed", result)
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
	return core.Ok(output)
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
		Assignee:    stringValue(values["assignee"]),
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

func issueReportBody(report any) core.Result {
	switch value := report.(type) {
	case nil:
		return core.Ok("")
	case string:
		return core.Ok(core.Trim(value))
	}

	if text := stringValue(report); text != "" {
		return core.Ok(text)
	}

	if jsonText := core.JSONMarshalString(report); core.Trim(jsonText) != "" {
		return core.Ok(core.Concat("```json\n", jsonText, "\n```"))
	}

	return core.Ok("")
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
