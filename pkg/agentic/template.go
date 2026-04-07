// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"dappco.re/go/agent/pkg/lib"
	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gopkg.in/yaml.v3"
)

// variable := agentic.TemplateVariable{Name: "feature_name", Required: true}
type TemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// summary := agentic.TemplateSummary{Slug: "bug-fix", Name: "Bug Fix", PhasesCount: 6}
type TemplateSummary struct {
	Slug        string              `json:"slug"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Category    string              `json:"category,omitempty"`
	PhasesCount int                 `json:"phases_count"`
	Variables   []TemplateVariable  `json:"variables,omitempty"`
	Version     PlanTemplateVersion `json:"version"`
}

// input := agentic.TemplateListInput{Category: "development"}
type TemplateListInput struct {
	Category string `json:"category,omitempty"`
}

// out := agentic.TemplateListOutput{Success: true, Total: 2}
type TemplateListOutput struct {
	Success   bool              `json:"success"`
	Templates []TemplateSummary `json:"templates"`
	Total     int               `json:"total"`
}

// input := agentic.TemplatePreviewInput{Template: "new-feature", Variables: map[string]string{"feature_name": "Auth"}}
type TemplatePreviewInput struct {
	Template     string            `json:"template,omitempty"`
	TemplateSlug string            `json:"template_slug,omitempty"`
	Slug         string            `json:"slug,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
}

// out := agentic.TemplatePreviewOutput{Success: true, Template: "bug-fix"}
type TemplatePreviewOutput struct {
	Success  bool                `json:"success"`
	Template string              `json:"template"`
	Version  PlanTemplateVersion `json:"version"`
	Preview  string              `json:"preview"`
}

// input := agentic.TemplateCreatePlanInput{Template: "new-feature", Variables: map[string]string{"feature_name": "Auth"}}
type TemplateCreatePlanInput struct {
	Template     string            `json:"template,omitempty"`
	TemplateSlug string            `json:"template_slug,omitempty"`
	Variables    map[string]string `json:"variables,omitempty"`
	Slug         string            `json:"slug,omitempty"`
	Title        string            `json:"title,omitempty"`
	Activate     bool              `json:"activate,omitempty"`
}

// out := agentic.TemplateCreatePlanOutput{Success: true, Plan: agentic.PlanCompatibilitySummary{Slug: "auth-feature"}}
type TemplateCreatePlanOutput struct {
	Success  bool                     `json:"success"`
	Plan     PlanCompatibilitySummary `json:"plan"`
	Version  PlanTemplateVersion      `json:"version"`
	Commands map[string]string        `json:"commands,omitempty"`
}

// version := agentic.PlanTemplateVersion{Slug: "new-feature", Version: 1, ContentHash: "..." }
type PlanTemplateVersion struct {
	Slug        string                 `json:"slug"`
	Version     int                    `json:"version"`
	Name        string                 `json:"name"`
	Content     planTemplateDefinition `json:"content,omitempty"`
	ContentHash string                 `json:"content_hash"`
}

type planTemplateDefinition struct {
	Slug        string                             `yaml:"-"`
	Name        string                             `yaml:"name"`
	Description string                             `yaml:"description"`
	Category    string                             `yaml:"category"`
	Variables   map[string]planTemplateVariableDef `yaml:"variables"`
	Guidelines  []string                           `yaml:"guidelines"`
	Phases      []planTemplatePhaseDef             `yaml:"phases"`
}

type planTemplateVariableDef struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type planTemplatePhaseDef struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tasks       []any  `yaml:"tasks"`
}

// result := c.Action("template.list").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) handleTemplateList(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.templateList(ctx, nil, TemplateListInput{
		Category: optionStringValue(options, "category"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("template.preview").Run(ctx, core.NewOptions(core.Option{Key: "template", Value: "bug-fix"}))
func (s *PrepSubsystem) handleTemplatePreview(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.templatePreview(ctx, nil, TemplatePreviewInput{
		Template:     optionStringValue(options, "template"),
		TemplateSlug: optionStringValue(options, "template_slug", "template-slug", "slug"),
		Variables:    optionStringMapValue(options, "variables"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

// result := c.Action("template.create_plan").Run(ctx, core.NewOptions(core.Option{Key: "template", Value: "bug-fix"}))
func (s *PrepSubsystem) handleTemplateCreatePlan(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.templateCreatePlan(ctx, nil, TemplateCreatePlanInput{
		Template:     optionStringValue(options, "template"),
		TemplateSlug: optionStringValue(options, "template_slug", "template-slug"),
		Variables:    optionStringMapValue(options, "variables"),
		Slug:         optionStringValue(options, "slug", "plan_slug", "plan-slug"),
		Title:        optionStringValue(options, "title"),
		Activate:     optionBoolValue(options, "activate"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerTemplateTools(svc *coremcp.Service) {
	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "template_list",
		Description: "List available plan templates with variables, category, and phase counts.",
	}, s.templateList)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "template_preview",
		Description: "Preview a plan template with variable substitution before creating a stored plan.",
	}, s.templatePreview)

	mcp.AddTool(svc.Server(), &mcp.Tool{
		Name:        "template_create_plan",
		Description: "Create a stored plan from an embedded YAML template, with optional activation.",
	}, s.templateCreatePlan)
}

func (s *PrepSubsystem) templateList(_ context.Context, _ *mcp.CallToolRequest, input TemplateListInput) (*mcp.CallToolResult, TemplateListOutput, error) {
	templates := make([]TemplateSummary, 0, len(lib.ListTasks()))
	for _, slug := range lib.ListTasks() {
		definition, version, err := loadPlanTemplateDefinition(slug, nil)
		if err != nil {
			continue
		}
		if input.Category != "" && definition.Category != input.Category {
			continue
		}
		templates = append(templates, templateSummaryFromDefinition(definition, version))
	}

	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Slug < templates[j].Slug
	})

	return nil, TemplateListOutput{
		Success:   true,
		Templates: templates,
		Total:     len(templates),
	}, nil
}

func (s *PrepSubsystem) templatePreview(_ context.Context, _ *mcp.CallToolRequest, input TemplatePreviewInput) (*mcp.CallToolResult, TemplatePreviewOutput, error) {
	templateName := templateNameValue(input.Template, input.TemplateSlug, input.Slug)
	if templateName == "" {
		return nil, TemplatePreviewOutput{}, core.E("templatePreview", "template is required", nil)
	}

	definition, version, err := loadPlanTemplateDefinition(templateName, input.Variables)
	if err != nil {
		return nil, TemplatePreviewOutput{}, err
	}

	return nil, TemplatePreviewOutput{
		Success:  true,
		Template: definition.Slug,
		Version:  version,
		Preview:  renderPlanMarkdown(definition, ""),
	}, nil
}

func (s *PrepSubsystem) templateCreatePlan(ctx context.Context, _ *mcp.CallToolRequest, input TemplateCreatePlanInput) (*mcp.CallToolResult, TemplateCreatePlanOutput, error) {
	templateName := templateNameValue(input.Template, input.TemplateSlug)
	if templateName == "" {
		return nil, TemplateCreatePlanOutput{}, core.E("templateCreatePlan", "template is required", nil)
	}

	definition, version, err := loadPlanTemplateDefinition(templateName, input.Variables)
	if err != nil {
		return nil, TemplateCreatePlanOutput{}, err
	}

	if missing := missingTemplateVariables(definition, input.Variables); len(missing) > 0 {
		return nil, TemplateCreatePlanOutput{}, core.E("templateCreatePlan", core.Concat("missing required variables: ", core.Join(", ", missing...)), nil)
	}

	title := core.Trim(input.Title)
	if title == "" {
		title = definition.Name
	}

	contextData := map[string]any{
		"template": definition.Slug,
	}
	if len(input.Variables) > 0 {
		contextData["variables"] = input.Variables
	}

	_, created, err := s.planCreate(ctx, nil, PlanCreateInput{
		Title:           title,
		Slug:            input.Slug,
		Objective:       definition.Description,
		Description:     definition.Description,
		Context:         contextData,
		TemplateVersion: version,
		Phases:          templatePlanPhases(definition),
		Notes:           templateGuidelinesNote(definition.Guidelines),
	})
	if err != nil {
		return nil, TemplateCreatePlanOutput{}, err
	}

	plan, err := readPlan(PlansRoot(), created.ID)
	if err != nil {
		return nil, TemplateCreatePlanOutput{}, err
	}

	if input.Activate {
		_, updated, updateErr := s.planUpdate(ctx, nil, PlanUpdateInput{
			Slug:   plan.Slug,
			Status: planCompatibilityInputStatus("active"),
		})
		if updateErr != nil {
			return nil, TemplateCreatePlanOutput{}, updateErr
		}
		plan = &updated.Plan
	}

	return nil, TemplateCreatePlanOutput{
		Success: true,
		Plan:    planCompatibilitySummary(*plan),
		Version: version,
	}, nil
}

func templateNameValue(values ...string) string {
	for _, value := range values {
		if trimmed := core.Trim(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func loadPlanTemplateDefinition(slug string, variables map[string]string) (planTemplateDefinition, PlanTemplateVersion, error) {
	result := lib.Task(slug)
	if !result.OK {
		err, _ := result.Value.(error)
		if err == nil {
			err = core.E("loadPlanTemplateDefinition", core.Concat("template not found: ", slug), nil)
		}
		return planTemplateDefinition{}, PlanTemplateVersion{}, err
	}

	rawContent := result.Value.(string)
	content := applyTemplateVariables(rawContent, variables)

	var definition planTemplateDefinition
	if err := yaml.Unmarshal([]byte(content), &definition); err != nil {
		return planTemplateDefinition{}, PlanTemplateVersion{}, core.E("loadPlanTemplateDefinition", core.Concat("invalid plan template: ", slug), err)
	}
	if definition.Name == "" || len(definition.Phases) == 0 {
		return planTemplateDefinition{}, PlanTemplateVersion{}, core.E("loadPlanTemplateDefinition", core.Concat("invalid plan template: ", slug), nil)
	}

	definition.Slug = slug
	return definition, templateVersionFromContent(slug, definition.Name, rawContent), nil
}

func applyTemplateVariables(content string, variables map[string]string) string {
	for key, value := range variables {
		content = core.Replace(content, core.Concat("{{", key, "}}"), value)
		content = core.Replace(content, core.Concat("{{ ", key, " }}"), value)
	}
	return content
}

func renderPlanMarkdown(definition planTemplateDefinition, task string) string {
	plan := core.NewBuilder()
	plan.WriteString(core.Concat("# ", definition.Name, "\n\n"))
	if task != "" {
		plan.WriteString(core.Concat("**Task:** ", task, "\n\n"))
	}
	if definition.Description != "" {
		plan.WriteString(core.Concat(definition.Description, "\n\n"))
	}

	if len(definition.Guidelines) > 0 {
		plan.WriteString("## Guidelines\n\n")
		for _, guideline := range definition.Guidelines {
			plan.WriteString(core.Concat("- ", guideline, "\n"))
		}
		plan.WriteString("\n")
	}

	for index, phase := range definition.Phases {
		plan.WriteString(core.Sprintf("## Phase %d: %s\n\n", index+1, phase.Name))
		if phase.Description != "" {
			plan.WriteString(core.Concat(phase.Description, "\n\n"))
		}
		for _, taskItem := range templatePlanTasks(phase.Tasks) {
			plan.WriteString(core.Concat("- [ ] ", taskItem.Title, "\n"))
		}
		plan.WriteString("\n")
	}

	return plan.String()
}

func templateSummaryFromDefinition(definition planTemplateDefinition, version PlanTemplateVersion) TemplateSummary {
	return TemplateSummary{
		Slug:        definition.Slug,
		Name:        definition.Name,
		Description: definition.Description,
		Category:    definition.Category,
		PhasesCount: len(definition.Phases),
		Variables:   templateVariableList(definition),
		Version:     version,
	}
}

func templateVersionFromContent(slug, name, content string) PlanTemplateVersion {
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	version := templateVersionForHash(slug, hash)

	var snapshot planTemplateDefinition
	if err := yaml.Unmarshal([]byte(content), &snapshot); err != nil {
		snapshot = planTemplateDefinition{}
	}
	snapshot.Slug = slug
	if snapshot.Name == "" {
		snapshot.Name = name
	}

	return PlanTemplateVersion{
		Slug:        slug,
		Version:     version,
		Name:        name,
		Content:     snapshot,
		ContentHash: hash,
	}
}

func templateVersionForHash(slug, contentHash string) int {
	if core.Trim(slug) == "" {
		return 1
	}

	version := 0
	matchedVersion := 0
	for _, path := range core.PathGlob(core.JoinPath(PlansRoot(), "*.json")) {
		id := core.TrimSuffix(core.PathBase(path), ".json")
		planResult := readPlanResult(PlansRoot(), id)
		if !planResult.OK {
			continue
		}

		plan, ok := planResult.Value.(*Plan)
		if !ok || plan == nil {
			continue
		}
		if plan.TemplateVersion.Slug != slug || plan.TemplateVersion.Version <= 0 {
			continue
		}

		if plan.TemplateVersion.ContentHash == contentHash {
			if plan.TemplateVersion.Version > matchedVersion {
				matchedVersion = plan.TemplateVersion.Version
			}
		}
		if plan.TemplateVersion.Version > version {
			version = plan.TemplateVersion.Version
		}
	}

	if matchedVersion > 0 {
		return matchedVersion
	}
	if version > 0 {
		return version + 1
	}
	return 1
}

func templateVariableList(definition planTemplateDefinition) []TemplateVariable {
	if len(definition.Variables) == 0 {
		return nil
	}

	names := make([]string, 0, len(definition.Variables))
	for name := range definition.Variables {
		names = append(names, name)
	}
	sort.Strings(names)

	variables := make([]TemplateVariable, 0, len(names))
	for _, name := range names {
		definitionValue := definition.Variables[name]
		variables = append(variables, TemplateVariable{
			Name:        name,
			Description: definitionValue.Description,
			Required:    definitionValue.Required,
		})
	}
	return variables
}

func missingTemplateVariables(definition planTemplateDefinition, variables map[string]string) []string {
	var missing []string
	for name, variable := range definition.Variables {
		if !variable.Required {
			continue
		}
		if core.Trim(variables[name]) == "" {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	return missing
}

func templatePlanPhases(definition planTemplateDefinition) []Phase {
	phases := make([]Phase, 0, len(definition.Phases))
	for index, phaseDefinition := range definition.Phases {
		phases = append(phases, Phase{
			Number:      index + 1,
			Name:        phaseDefinition.Name,
			Description: phaseDefinition.Description,
			Status:      "pending",
			Tasks:       templatePlanTasks(phaseDefinition.Tasks),
		})
	}
	return phases
}

func templatePlanTasks(items []any) []PlanTask {
	tasks := make([]PlanTask, 0, len(items))
	for index, item := range items {
		task := templatePlanTask(item, index+1)
		if task.Title == "" {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func templatePlanTask(item any, number int) PlanTask {
	switch value := item.(type) {
	case string:
		return PlanTask{
			ID:     core.Sprint(number),
			Title:  core.Trim(value),
			Status: "pending",
		}
	case map[string]any:
		title := stringValue(value["title"])
		if title == "" {
			title = stringValue(value["name"])
		}
		status := stringValue(value["status"])
		if status == "" {
			status = "pending"
		}
		taskID := stringValue(value["id"])
		if taskID == "" {
			taskID = core.Sprint(number)
		}
		return PlanTask{
			ID:          taskID,
			Title:       title,
			Description: stringValue(value["description"]),
			Status:      status,
			Notes:       stringValue(value["notes"]),
			File:        stringValue(value["file"]),
			Line:        intValue(value["line"]),
		}
	}
	return PlanTask{}
}

func templateGuidelinesNote(guidelines []string) string {
	cleaned := cleanStrings(guidelines)
	if len(cleaned) == 0 {
		return ""
	}

	builder := core.NewBuilder()
	builder.WriteString("Guidelines:\n")
	for _, guideline := range cleaned {
		builder.WriteString(core.Concat("- ", guideline, "\n"))
	}
	return core.TrimSuffix(builder.String(), "\n")
}
