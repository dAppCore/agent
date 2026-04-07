// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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

// input := agentic.ContentSchemaInput{Type: "article", Title: "Release notes", URL: "https://example.test/releases"}
type ContentSchemaInput struct {
	Type        string                  `json:"type,omitempty"`
	Title       string                  `json:"title,omitempty"`
	Description string                  `json:"description,omitempty"`
	URL         string                  `json:"url,omitempty"`
	Author      string                  `json:"author,omitempty"`
	PublishedAt string                  `json:"published_at,omitempty"`
	ModifiedAt  string                  `json:"modified_at,omitempty"`
	Language    string                  `json:"language,omitempty"`
	Image       string                  `json:"image,omitempty"`
	Questions   []ContentSchemaQuestion `json:"questions,omitempty"`
	Steps       []ContentSchemaStep     `json:"steps,omitempty"`
}

// question := agentic.ContentSchemaQuestion{Question: "What changed?", Answer: "The release notes are now generated from plans."}
type ContentSchemaQuestion struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// step := agentic.ContentSchemaStep{Name: "Review", Text: "Check the draft for accuracy."}
type ContentSchemaStep struct {
	Name string `json:"name,omitempty"`
	Text string `json:"text,omitempty"`
	URL  string `json:"url,omitempty"`
}

// out := agentic.ContentSchemaOutput{Success: true, SchemaType: "FAQPage"}
type ContentSchemaOutput struct {
	Success    bool           `json:"success"`
	SchemaType string         `json:"schema_type"`
	SchemaJSON string         `json:"schema_json"`
	Schema     map[string]any `json:"schema"`
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

func (s *PrepSubsystem) contentGenerateResult(ctx context.Context, input ContentGenerateInput) (ContentResult, error) {
	if err := s.validateContentProvider(input.Provider); err != nil {
		return ContentResult{}, err
	}

	hasPrompt := core.Trim(input.Prompt) != ""
	hasBrief := core.Trim(input.BriefID) != ""
	hasTemplate := core.Trim(input.Template) != ""
	if !hasPrompt && !(hasBrief && hasTemplate) {
		return ContentResult{}, core.E("contentGenerate", "prompt or brief_id plus template is required", nil)
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
		return ContentResult{}, resultErrorValue("content.generate", result)
	}

	return parseContentResult(payloadResourceMap(result.Value.(map[string]any), "result", "content", "generation")), nil
}

func (s *PrepSubsystem) validateContentProvider(providerName string) error {
	if core.Trim(providerName) == "" {
		return nil
	}

	provider, ok := s.providerManager().Provider(providerName)
	if !ok {
		return core.E("contentGenerate", core.Concat("unknown provider: ", providerName), nil)
	}
	if !provider.IsAvailable() {
		return core.E("contentGenerate", core.Concat("provider unavailable: ", providerName), nil)
	}

	return nil
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

// result := c.Action("content.schema.generate").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "type", Value: "howto"},
//	core.Option{Key: "title", Value: "Set up the workspace"},
//
// ))
func (s *PrepSubsystem) handleContentSchemaGenerate(ctx context.Context, options core.Options) core.Result {
	input := ContentSchemaInput{
		Type:        optionStringValue(options, "schema_type", "schema-type", "type", "kind"),
		Title:       optionStringValue(options, "title", "headline"),
		Description: optionStringValue(options, "description"),
		URL:         optionStringValue(options, "url", "link"),
		Author:      optionStringValue(options, "author"),
		PublishedAt: optionStringValue(options, "published_at", "published-at", "date_published"),
		ModifiedAt:  optionStringValue(options, "modified_at", "modified-at", "date_modified"),
		Language:    optionStringValue(options, "language", "in_language", "in-language"),
		Image:       optionStringValue(options, "image", "image_url", "image-url"),
	}
	if value := optionAnyValue(options, "questions", "faq"); value != nil {
		input.Questions = contentSchemaQuestionsValue(value)
	}
	if value := optionAnyValue(options, "steps"); value != nil {
		input.Steps = contentSchemaStepsValue(value)
	}

	_, output, err := s.contentSchemaGenerate(ctx, nil, input)
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerContentTools(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_generate",
		Description: "Generate content from a prompt or a brief/template pair using the platform AI provider abstraction.",
	}, s.contentGenerate)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_batch_generate",
		Description: "Generate content for a stored batch specification.",
	}, s.contentBatchGenerate)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_batch",
		Description: "Generate content for a stored batch specification using the legacy MCP alias.",
	}, s.contentBatchGenerate)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_brief_create",
		Description: "Create a reusable content brief for later generation work.",
	}, s.contentBriefCreate)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_brief_get",
		Description: "Read a reusable content brief by ID or slug.",
	}, s.contentBriefGet)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_brief_list",
		Description: "List reusable content briefs with optional category and product filters.",
	}, s.contentBriefList)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_status",
		Description: "Read batch content generation status by batch ID.",
	}, s.contentStatus)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_usage_stats",
		Description: "Read AI usage statistics for the content pipeline.",
	}, s.contentUsageStats)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_from_plan",
		Description: "Generate content using stored plan context and an optional provider override.",
	}, s.contentFromPlan)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "content_schema_generate",
		Description: "Generate SEO schema JSON-LD for article, FAQ, or how-to content.",
	}, s.contentSchemaGenerate)
}

func (s *PrepSubsystem) contentGenerate(ctx context.Context, _ *mcp.CallToolRequest, input ContentGenerateInput) (*mcp.CallToolResult, ContentGenerateOutput, error) {
	content, err := s.contentGenerateResult(ctx, input)
	if err != nil {
		return nil, ContentGenerateOutput{}, err
	}
	return nil, ContentGenerateOutput{
		Success: true,
		Result:  content,
	}, nil
}

func (s *PrepSubsystem) contentBatchGenerate(ctx context.Context, _ *mcp.CallToolRequest, input ContentBatchGenerateInput) (*mcp.CallToolResult, ContentBatchOutput, error) {
	if core.Trim(input.BatchID) == "" {
		return nil, ContentBatchOutput{}, core.E("contentBatchGenerate", "batch_id is required", nil)
	}
	if err := s.validateContentProvider(input.Provider); err != nil {
		return nil, ContentBatchOutput{}, err
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
	if err := s.validateContentProvider(input.Provider); err != nil {
		return nil, ContentFromPlanOutput{}, err
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

func (s *PrepSubsystem) contentSchemaGenerate(_ context.Context, _ *mcp.CallToolRequest, input ContentSchemaInput) (*mcp.CallToolResult, ContentSchemaOutput, error) {
	schemaType := contentSchemaType(input.Type)
	if schemaType == "" {
		return nil, ContentSchemaOutput{}, core.E("contentSchemaGenerate", "schema type is required", nil)
	}
	if core.Trim(input.Title) == "" {
		return nil, ContentSchemaOutput{}, core.E("contentSchemaGenerate", "title is required", nil)
	}

	schema := map[string]any{
		"@context": "https://schema.org",
		"@type":    schemaType,
		"name":     input.Title,
		"headline": input.Title,
	}
	if input.Description != "" {
		schema["description"] = input.Description
	}
	if input.URL != "" {
		schema["url"] = input.URL
		schema["mainEntityOfPage"] = input.URL
	}
	if input.Author != "" {
		schema["author"] = map[string]any{
			"@type": "Person",
			"name":  input.Author,
		}
	}
	if input.PublishedAt != "" {
		schema["datePublished"] = input.PublishedAt
	}
	if input.ModifiedAt != "" {
		schema["dateModified"] = input.ModifiedAt
	}
	if input.Language != "" {
		schema["inLanguage"] = input.Language
	}
	if input.Image != "" {
		schema["image"] = input.Image
	}

	switch schemaType {
	case "FAQPage":
		if len(input.Questions) == 0 {
			return nil, ContentSchemaOutput{}, core.E("contentSchemaGenerate", "questions are required for FAQ schema", nil)
		}
		schema["mainEntity"] = contentSchemaFAQEntries(input.Questions)
	case "HowTo":
		if len(input.Steps) == 0 {
			return nil, ContentSchemaOutput{}, core.E("contentSchemaGenerate", "steps are required for how-to schema", nil)
		}
		schema["step"] = contentSchemaHowToSteps(input.Steps)
	case "BlogPosting", "TechArticle":
		if len(input.Steps) > 0 {
			schema["step"] = contentSchemaHowToSteps(input.Steps)
		}
		if len(input.Questions) > 0 {
			schema["mainEntity"] = contentSchemaFAQEntries(input.Questions)
		}
	default:
		if len(input.Questions) > 0 {
			schema["mainEntity"] = contentSchemaFAQEntries(input.Questions)
		}
		if len(input.Steps) > 0 {
			schema["step"] = contentSchemaHowToSteps(input.Steps)
		}
	}

	return nil, ContentSchemaOutput{
		Success:    true,
		SchemaType: schemaType,
		SchemaJSON: core.JSONMarshalString(schema),
		Schema:     schema,
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

func contentSchemaType(value string) string {
	switch core.Lower(core.Trim(value)) {
	case "article", "blog", "blogpost", "blog-post", "blogposting":
		return "BlogPosting"
	case "tech", "techarticle", "tech-article":
		return "TechArticle"
	case "faq", "faqpage":
		return "FAQPage"
	case "howto", "how-to":
		return "HowTo"
	}
	return ""
}

func contentSchemaQuestionsValue(value any) []ContentSchemaQuestion {
	items := contentSchemaItemsValue(value)
	if len(items) == 0 {
		return nil
	}
	questions := make([]ContentSchemaQuestion, 0, len(items))
	for _, item := range items {
		question := contentMapStringValue(item, "question", "name", "title")
		answer := contentMapStringValue(item, "answer", "text", "body", "content")
		if question == "" || answer == "" {
			continue
		}
		questions = append(questions, ContentSchemaQuestion{
			Question: question,
			Answer:   answer,
		})
	}
	return questions
}

func contentSchemaStepsValue(value any) []ContentSchemaStep {
	items := contentSchemaItemsValue(value)
	if len(items) == 0 {
		return nil
	}
	steps := make([]ContentSchemaStep, 0, len(items))
	for _, item := range items {
		text := contentMapStringValue(item, "text", "body", "content", "description")
		name := contentMapStringValue(item, "name", "title", "label")
		url := contentMapStringValue(item, "url", "link")
		if name == "" && text == "" {
			continue
		}
		steps = append(steps, ContentSchemaStep{
			Name: name,
			Text: text,
			URL:  url,
		})
	}
	return steps
}

func contentSchemaItemsValue(value any) []map[string]any {
	switch typed := value.(type) {
	case []ContentSchemaQuestion:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, map[string]any{
				"question": item.Question,
				"answer":   item.Answer,
			})
		}
		return items
	case []ContentSchemaStep:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, map[string]any{
				"name": item.Name,
				"text": item.Text,
				"url":  item.URL,
			})
		}
		return items
	case []map[string]any:
		return typed
	case []any:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if parsed := contentSchemaItemMap(item); len(parsed) > 0 {
				items = append(items, parsed)
			}
		}
		return items
	case map[string]any:
		return []map[string]any{typed}
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		var generic []any
		if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
			return contentSchemaItemsValue(generic)
		}
		var single map[string]any
		if result := core.JSONUnmarshalString(trimmed, &single); result.OK {
			return []map[string]any{single}
		}
	}
	return nil
}

func contentSchemaItemMap(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case ContentSchemaQuestion:
		return map[string]any{
			"question": typed.Question,
			"answer":   typed.Answer,
		}
	case ContentSchemaStep:
		return map[string]any{
			"name": typed.Name,
			"text": typed.Text,
			"url":  typed.URL,
		}
	case string:
		trimmed := core.Trim(typed)
		if trimmed == "" {
			return nil
		}
		var generic map[string]any
		if result := core.JSONUnmarshalString(trimmed, &generic); result.OK {
			return generic
		}
	}
	return nil
}

func contentSchemaFAQEntries(questions []ContentSchemaQuestion) []map[string]any {
	entries := make([]map[string]any, 0, len(questions))
	for _, question := range questions {
		if core.Trim(question.Question) == "" || core.Trim(question.Answer) == "" {
			continue
		}
		entries = append(entries, map[string]any{
			"@type": "Question",
			"name":  question.Question,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  question.Answer,
			},
		})
	}
	return entries
}

func contentSchemaHowToSteps(steps []ContentSchemaStep) []map[string]any {
	entries := make([]map[string]any, 0, len(steps))
	for _, step := range steps {
		if core.Trim(step.Name) == "" && core.Trim(step.Text) == "" {
			continue
		}
		entry := map[string]any{
			"@type": "HowToStep",
		}
		if step.Name != "" {
			entry["name"] = step.Name
		}
		if step.Text != "" {
			entry["text"] = step.Text
		}
		if step.URL != "" {
			entry["url"] = step.URL
		}
		entries = append(entries, entry)
	}
	return entries
}
