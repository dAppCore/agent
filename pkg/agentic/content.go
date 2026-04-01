// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// result := agentic.ContentResult{Provider: "claude", Model: "claude-3.7-sonnet", Content: "Draft ready"}
type ContentResult struct {
	ID           string         `json:"id,omitempty"`
	BatchID      string         `json:"batch_id,omitempty"`
	Prompt       string         `json:"prompt,omitempty"`
	Provider     string         `json:"provider,omitempty"`
	Model        string         `json:"model,omitempty"`
	Content      string         `json:"content,omitempty"`
	Status       string         `json:"status,omitempty"`
	InputTokens  int            `json:"input_tokens,omitempty"`
	OutputTokens int            `json:"output_tokens,omitempty"`
	DurationMS   int            `json:"duration_ms,omitempty"`
	StopReason   string         `json:"stop_reason,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Raw          map[string]any `json:"raw,omitempty"`
}

// brief := agentic.ContentBrief{ID: "brief_1", Slug: "host-link", Title: "LinkHost", Category: "product"}
type ContentBrief struct {
	ID        string         `json:"id,omitempty"`
	Slug      string         `json:"slug,omitempty"`
	Name      string         `json:"name,omitempty"`
	Title     string         `json:"title,omitempty"`
	Product   string         `json:"product,omitempty"`
	Category  string         `json:"category,omitempty"`
	Brief     string         `json:"brief,omitempty"`
	Summary   string         `json:"summary,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
}

//	input := agentic.ContentGenerateInput{
//	    BriefID:  "brief_1",
//	    Template: "help-article",
//	    Provider: "claude",
//	}
type ContentGenerateInput struct {
	Prompt   string         `json:"prompt,omitempty"`
	BriefID  string         `json:"brief_id,omitempty"`
	Template string         `json:"template,omitempty"`
	Provider string         `json:"provider,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
}

// input := agentic.ContentBatchGenerateInput{BatchID: "batch_123", Provider: "gemini"}
type ContentBatchGenerateInput struct {
	BatchID  string `json:"batch_id"`
	Provider string `json:"provider,omitempty"`
	DryRun   bool   `json:"dry_run,omitempty"`
}

// input := agentic.ContentBriefCreateInput{Title: "LinkHost brief", Product: "LinkHost"}
type ContentBriefCreateInput struct {
	Title    string         `json:"title,omitempty"`
	Name     string         `json:"name,omitempty"`
	Slug     string         `json:"slug,omitempty"`
	Product  string         `json:"product,omitempty"`
	Category string         `json:"category,omitempty"`
	Brief    string         `json:"brief,omitempty"`
	Summary  string         `json:"summary,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Context  map[string]any `json:"context,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// input := agentic.ContentBriefGetInput{BriefID: "host-link"}
type ContentBriefGetInput struct {
	BriefID string `json:"brief_id"`
}

// input := agentic.ContentBriefListInput{Category: "product", Limit: 10}
type ContentBriefListInput struct {
	Category string `json:"category,omitempty"`
	Product  string `json:"product,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// input := agentic.ContentStatusInput{BatchID: "batch_123"}
type ContentStatusInput struct {
	BatchID string `json:"batch_id"`
}

// input := agentic.ContentUsageStatsInput{Provider: "claude", Period: "week"}
type ContentUsageStatsInput struct {
	Provider string `json:"provider,omitempty"`
	Period   string `json:"period,omitempty"`
	Since    string `json:"since,omitempty"`
	Until    string `json:"until,omitempty"`
}

// input := agentic.ContentFromPlanInput{PlanSlug: "release-notes", Provider: "openai"}
type ContentFromPlanInput struct {
	PlanSlug string         `json:"plan_slug"`
	Provider string         `json:"provider,omitempty"`
	Prompt   string         `json:"prompt,omitempty"`
	Template string         `json:"template,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

// out := agentic.ContentGenerateOutput{Success: true, Result: agentic.ContentResult{Provider: "claude"}}
type ContentGenerateOutput struct {
	Success bool          `json:"success"`
	Result  ContentResult `json:"result"`
}

// out := agentic.ContentBatchOutput{Success: true, Batch: map[string]any{"batch_id": "batch_123"}}
type ContentBatchOutput struct {
	Success bool           `json:"success"`
	Batch   map[string]any `json:"batch"`
}

// out := agentic.ContentBriefOutput{Success: true, Brief: agentic.ContentBrief{Slug: "host-link"}}
type ContentBriefOutput struct {
	Success bool         `json:"success"`
	Brief   ContentBrief `json:"brief"`
}

// out := agentic.ContentBriefListOutput{Success: true, Total: 1, Briefs: []agentic.ContentBrief{{Slug: "host-link"}}}
type ContentBriefListOutput struct {
	Success bool           `json:"success"`
	Total   int            `json:"total"`
	Briefs  []ContentBrief `json:"briefs"`
}

// out := agentic.ContentStatusOutput{Success: true, Status: map[string]any{"status": "running"}}
type ContentStatusOutput struct {
	Success bool           `json:"success"`
	Status  map[string]any `json:"status"`
}

// out := agentic.ContentUsageStatsOutput{Success: true, Usage: map[string]any{"calls": 4}}
type ContentUsageStatsOutput struct {
	Success bool           `json:"success"`
	Usage   map[string]any `json:"usage"`
}

// out := agentic.ContentFromPlanOutput{Success: true, Result: agentic.ContentResult{BatchID: "batch_123"}}
type ContentFromPlanOutput struct {
	Success bool          `json:"success"`
	Result  ContentResult `json:"result"`
}

// result := c.Action("content.generate").Run(ctx, core.NewOptions(core.Option{Key: "prompt", Value: "Draft a release note"}))
func (s *PrepSubsystem) handleContentGenerate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentGenerate(ctx, nil, ContentGenerateInput{
		Prompt:   optionStringValue(options, "prompt"),
		BriefID:  optionStringValue(options, "brief_id", "brief-id"),
		Template: optionStringValue(options, "template"),
		Provider: optionStringValue(options, "provider"),
		Config:   optionAnyMapValue(options, "config"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.batch.generate").Run(ctx, core.NewOptions(core.Option{Key: "batch_id", Value: "batch_123"}))
func (s *PrepSubsystem) handleContentBatchGenerate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentBatchGenerate(ctx, nil, ContentBatchGenerateInput{
		BatchID:  optionStringValue(options, "batch_id", "batch-id", "_arg"),
		Provider: optionStringValue(options, "provider"),
		DryRun:   optionBoolValue(options, "dry_run", "dry-run"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.brief.create").Run(ctx, core.NewOptions(core.Option{Key: "title", Value: "LinkHost brief"}))
func (s *PrepSubsystem) handleContentBriefCreate(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentBriefCreate(ctx, nil, ContentBriefCreateInput{
		Title:    optionStringValue(options, "title"),
		Name:     optionStringValue(options, "name"),
		Slug:     optionStringValue(options, "slug"),
		Product:  optionStringValue(options, "product"),
		Category: optionStringValue(options, "category"),
		Brief:    optionStringValue(options, "brief"),
		Summary:  optionStringValue(options, "summary"),
		Metadata: optionAnyMapValue(options, "metadata"),
		Context:  optionAnyMapValue(options, "context"),
		Payload:  optionAnyMapValue(options, "payload"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.brief.get").Run(ctx, core.NewOptions(core.Option{Key: "brief_id", Value: "host-link"}))
func (s *PrepSubsystem) handleContentBriefGet(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentBriefGet(ctx, nil, ContentBriefGetInput{
		BriefID: optionStringValue(options, "brief_id", "brief-id", "id", "slug", "_arg"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.brief.list").Run(ctx, core.NewOptions(core.Option{Key: "category", Value: "product"}))
func (s *PrepSubsystem) handleContentBriefList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentBriefList(ctx, nil, ContentBriefListInput{
		Category: optionStringValue(options, "category"),
		Product:  optionStringValue(options, "product"),
		Limit:    optionIntValue(options, "limit"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.status").Run(ctx, core.NewOptions(core.Option{Key: "batch_id", Value: "batch_123"}))
func (s *PrepSubsystem) handleContentStatus(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentStatus(ctx, nil, ContentStatusInput{
		BatchID: optionStringValue(options, "batch_id", "batch-id", "_arg"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.usage.stats").Run(ctx, core.NewOptions(core.Option{Key: "provider", Value: "claude"}))
func (s *PrepSubsystem) handleContentUsageStats(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentUsageStats(ctx, nil, ContentUsageStatsInput{
		Provider: optionStringValue(options, "provider"),
		Period:   optionStringValue(options, "period"),
		Since:    optionStringValue(options, "since"),
		Until:    optionStringValue(options, "until"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("content.from.plan").Run(ctx, core.NewOptions(core.Option{Key: "plan_slug", Value: "release-notes"}))
func (s *PrepSubsystem) handleContentFromPlan(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.contentFromPlan(ctx, nil, ContentFromPlanInput{
		PlanSlug: optionStringValue(options, "plan_slug", "plan", "slug", "_arg"),
		Provider: optionStringValue(options, "provider"),
		Prompt:   optionStringValue(options, "prompt"),
		Template: optionStringValue(options, "template"),
		Config:   optionAnyMapValue(options, "config"),
		Payload:  optionAnyMapValue(options, "payload"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerContentTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_generate",
		Description: "Generate content from a prompt or a brief/template pair using the platform AI provider abstraction.",
	}, s.contentGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_batch_generate",
		Description: "Generate content for a stored batch specification.",
	}, s.contentBatchGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_batch",
		Description: "Generate content for a stored batch specification using the legacy MCP alias.",
	}, s.contentBatchGenerate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_brief_create",
		Description: "Create a reusable content brief for later generation work.",
	}, s.contentBriefCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_brief_get",
		Description: "Read a reusable content brief by ID or slug.",
	}, s.contentBriefGet)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_brief_list",
		Description: "List reusable content briefs with optional category and product filters.",
	}, s.contentBriefList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_status",
		Description: "Read batch content generation status by batch ID.",
	}, s.contentStatus)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_usage_stats",
		Description: "Read AI usage statistics for the content pipeline.",
	}, s.contentUsageStats)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "content_from_plan",
		Description: "Generate content using stored plan context and an optional provider override.",
	}, s.contentFromPlan)
}

func (s *PrepSubsystem) contentGenerate(ctx context.Context, _ *mcp.CallToolRequest, input ContentGenerateInput) (*mcp.CallToolResult, ContentGenerateOutput, error) {
	hasPrompt := core.Trim(input.Prompt) != ""
	hasBrief := core.Trim(input.BriefID) != ""
	hasTemplate := core.Trim(input.Template) != ""
	if !hasPrompt && !(hasBrief && hasTemplate) {
		return nil, ContentGenerateOutput{}, core.E("contentGenerate", "prompt or brief_id plus template is required", nil)
	}

	body := map[string]any{}
	if hasPrompt {
		body["prompt"] = input.Prompt
	}
	if input.BriefID != "" {
		body["brief_id"] = input.BriefID
	}
	if input.Template != "" {
		body["template"] = input.Template
	}
	if input.Provider != "" {
		body["provider"] = input.Provider
	}
	if len(input.Config) > 0 {
		body["config"] = input.Config
	}

	result := s.platformPayload(ctx, "content.generate", "POST", "/v1/content/generate", body)
	if !result.OK {
		return nil, ContentGenerateOutput{}, resultErrorValue("content.generate", result)
	}

	return nil, ContentGenerateOutput{
		Success: true,
		Result:  parseContentResult(payloadResourceMap(result.Value.(map[string]any), "result", "content", "generation")),
	}, nil
}

func (s *PrepSubsystem) contentBatchGenerate(ctx context.Context, _ *mcp.CallToolRequest, input ContentBatchGenerateInput) (*mcp.CallToolResult, ContentBatchOutput, error) {
	if core.Trim(input.BatchID) == "" {
		return nil, ContentBatchOutput{}, core.E("contentBatchGenerate", "batch_id is required", nil)
	}

	body := map[string]any{
		"batch_id": input.BatchID,
	}
	if input.Provider != "" {
		body["provider"] = input.Provider
	}
	if input.DryRun {
		body["dry_run"] = true
	}

	result := s.platformPayload(ctx, "content.batch.generate", "POST", "/v1/content/batch/generate", body)
	if !result.OK {
		return nil, ContentBatchOutput{}, resultErrorValue("content.batch.generate", result)
	}

	return nil, ContentBatchOutput{
		Success: true,
		Batch:   payloadResourceMap(result.Value.(map[string]any), "batch", "result", "status"),
	}, nil
}

func (s *PrepSubsystem) contentBriefCreate(ctx context.Context, _ *mcp.CallToolRequest, input ContentBriefCreateInput) (*mcp.CallToolResult, ContentBriefOutput, error) {
	body := map[string]any{}
	if input.Title != "" {
		body["title"] = input.Title
	}
	if input.Name != "" {
		body["name"] = input.Name
	}
	if input.Slug != "" {
		body["slug"] = input.Slug
	}
	if input.Product != "" {
		body["product"] = input.Product
	}
	if input.Category != "" {
		body["category"] = input.Category
	}
	if input.Brief != "" {
		body["brief"] = input.Brief
	}
	if input.Summary != "" {
		body["summary"] = input.Summary
	}
	if len(input.Metadata) > 0 {
		body["metadata"] = input.Metadata
	}
	if len(input.Context) > 0 {
		body["context"] = input.Context
	}
	body = mergeContentPayload(body, input.Payload)
	if len(body) == 0 {
		return nil, ContentBriefOutput{}, core.E("contentBriefCreate", "content brief data is required", nil)
	}

	result := s.platformPayload(ctx, "content.brief.create", "POST", "/v1/content/briefs", body)
	if !result.OK {
		return nil, ContentBriefOutput{}, resultErrorValue("content.brief.create", result)
	}

	return nil, ContentBriefOutput{
		Success: true,
		Brief:   parseContentBrief(payloadResourceMap(result.Value.(map[string]any), "brief", "item")),
	}, nil
}

func (s *PrepSubsystem) contentBriefGet(ctx context.Context, _ *mcp.CallToolRequest, input ContentBriefGetInput) (*mcp.CallToolResult, ContentBriefOutput, error) {
	if core.Trim(input.BriefID) == "" {
		return nil, ContentBriefOutput{}, core.E("contentBriefGet", "brief_id is required", nil)
	}

	result := s.platformPayload(ctx, "content.brief.get", "GET", core.Concat("/v1/content/briefs/", input.BriefID), nil)
	if !result.OK {
		return nil, ContentBriefOutput{}, resultErrorValue("content.brief.get", result)
	}

	return nil, ContentBriefOutput{
		Success: true,
		Brief:   parseContentBrief(payloadResourceMap(result.Value.(map[string]any), "brief", "item")),
	}, nil
}

func (s *PrepSubsystem) contentBriefList(ctx context.Context, _ *mcp.CallToolRequest, input ContentBriefListInput) (*mcp.CallToolResult, ContentBriefListOutput, error) {
	path := "/v1/content/briefs"
	path = appendQueryParam(path, "category", input.Category)
	path = appendQueryParam(path, "product", input.Product)
	if input.Limit > 0 {
		path = appendQueryParam(path, "limit", core.Sprint(input.Limit))
	}

	result := s.platformPayload(ctx, "content.brief.list", "GET", path, nil)
	if !result.OK {
		return nil, ContentBriefListOutput{}, resultErrorValue("content.brief.list", result)
	}

	return nil, parseContentBriefListOutput(result.Value.(map[string]any)), nil
}

func (s *PrepSubsystem) contentStatus(ctx context.Context, _ *mcp.CallToolRequest, input ContentStatusInput) (*mcp.CallToolResult, ContentStatusOutput, error) {
	if core.Trim(input.BatchID) == "" {
		return nil, ContentStatusOutput{}, core.E("contentStatus", "batch_id is required", nil)
	}

	result := s.platformPayload(ctx, "content.status", "GET", core.Concat("/v1/content/status/", input.BatchID), nil)
	if !result.OK {
		return nil, ContentStatusOutput{}, resultErrorValue("content.status", result)
	}

	return nil, ContentStatusOutput{
		Success: true,
		Status:  payloadResourceMap(result.Value.(map[string]any), "status", "batch"),
	}, nil
}

func (s *PrepSubsystem) contentUsageStats(ctx context.Context, _ *mcp.CallToolRequest, input ContentUsageStatsInput) (*mcp.CallToolResult, ContentUsageStatsOutput, error) {
	path := "/v1/content/usage/stats"
	path = appendQueryParam(path, "provider", input.Provider)
	path = appendQueryParam(path, "period", input.Period)
	path = appendQueryParam(path, "since", input.Since)
	path = appendQueryParam(path, "until", input.Until)

	result := s.platformPayload(ctx, "content.usage.stats", "GET", path, nil)
	if !result.OK {
		return nil, ContentUsageStatsOutput{}, resultErrorValue("content.usage.stats", result)
	}

	return nil, ContentUsageStatsOutput{
		Success: true,
		Usage:   payloadResourceMap(result.Value.(map[string]any), "usage", "stats"),
	}, nil
}

func (s *PrepSubsystem) contentFromPlan(ctx context.Context, _ *mcp.CallToolRequest, input ContentFromPlanInput) (*mcp.CallToolResult, ContentFromPlanOutput, error) {
	if core.Trim(input.PlanSlug) == "" {
		return nil, ContentFromPlanOutput{}, core.E("contentFromPlan", "plan_slug is required", nil)
	}

	body := map[string]any{
		"plan_slug": input.PlanSlug,
	}
	if input.Provider != "" {
		body["provider"] = input.Provider
	}
	if input.Prompt != "" {
		body["prompt"] = input.Prompt
	}
	if input.Template != "" {
		body["template"] = input.Template
	}
	if len(input.Config) > 0 {
		body["config"] = input.Config
	}
	body = mergeContentPayload(body, input.Payload)

	result := s.platformPayload(ctx, "content.from.plan", "POST", "/v1/content/from-plan", body)
	if !result.OK {
		return nil, ContentFromPlanOutput{}, resultErrorValue("content.from.plan", result)
	}

	return nil, ContentFromPlanOutput{
		Success: true,
		Result:  parseContentResult(payloadResourceMap(result.Value.(map[string]any), "result", "content", "generation")),
	}, nil
}

func mergeContentPayload(target, extra map[string]any) map[string]any {
	if len(target) == 0 {
		target = map[string]any{}
	}
	for key, value := range extra {
		if value != nil {
			target[key] = value
		}
	}
	return target
}

func parseContentResult(values map[string]any) ContentResult {
	result := ContentResult{
		ID:           contentMapStringValue(values, "id"),
		BatchID:      contentMapStringValue(values, "batch_id", "batchId"),
		Prompt:       contentMapStringValue(values, "prompt"),
		Provider:     contentMapStringValue(values, "provider"),
		Model:        contentMapStringValue(values, "model", "model_name"),
		Content:      contentMapStringValue(values, "content", "text", "output"),
		Status:       contentMapStringValue(values, "status"),
		InputTokens:  mapIntValue(values, "input_tokens", "inputTokens"),
		OutputTokens: mapIntValue(values, "output_tokens", "outputTokens"),
		DurationMS:   mapIntValue(values, "duration_ms", "durationMs", "duration"),
		StopReason:   contentMapStringValue(values, "stop_reason", "stopReason"),
		Metadata:     anyMapValue(values["metadata"]),
		Raw:          anyMapValue(values["raw"]),
	}
	if len(result.Raw) == 0 && len(values) > 0 {
		result.Raw = values
	}
	return result
}

func parseContentBrief(values map[string]any) ContentBrief {
	return ContentBrief{
		ID:        contentMapStringValue(values, "id"),
		Slug:      contentMapStringValue(values, "slug"),
		Name:      contentMapStringValue(values, "name"),
		Title:     contentMapStringValue(values, "title"),
		Product:   contentMapStringValue(values, "product"),
		Category:  contentMapStringValue(values, "category"),
		Brief:     contentMapStringValue(values, "brief", "content", "body"),
		Summary:   contentMapStringValue(values, "summary", "description"),
		Metadata:  anyMapValue(values["metadata"]),
		CreatedAt: contentMapStringValue(values, "created_at"),
		UpdatedAt: contentMapStringValue(values, "updated_at"),
	}
}

func parseContentBriefListOutput(payload map[string]any) ContentBriefListOutput {
	values := payloadDataSlice(payload, "briefs", "items")
	briefs := make([]ContentBrief, 0, len(values))
	for _, value := range values {
		briefs = append(briefs, parseContentBrief(value))
	}

	total := mapIntValue(payload, "total", "count")
	if total == 0 {
		total = mapIntValue(payloadDataMap(payload), "total", "count")
	}
	if total == 0 {
		total = len(briefs)
	}

	return ContentBriefListOutput{
		Success: true,
		Total:   total,
		Briefs:  briefs,
	}
}

func contentMapStringValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			if text := stringValue(value); text != "" {
				return text
			}
		}
	}
	return ""
}
