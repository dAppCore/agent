// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// issue := agentic.Issue{Slug: "fix-auth", Title: "Fix auth middleware"}
// result := c.Action("plan.from.issue").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
type PlanFromIssueInput struct {
	ID   string `json:"id,omitempty"`
	Slug string `json:"slug,omitempty"`
}

// output := agentic.PlanFromIssueOutput{Success: true}
type PlanFromIssueOutput struct {
	Success bool   `json:"success"`
	Issue   Issue  `json:"issue"`
	Plan    Plan   `json:"plan"`
	Path    string `json:"path,omitempty"`
}

// result := c.Action("plan.from.issue").Run(ctx, core.NewOptions(core.Option{Key: "slug", Value: "fix-auth"}))
func (s *PrepSubsystem) handlePlanFromIssue(ctx context.Context, options core.Options) core.Result {
	_, output, err := s.planFromIssue(ctx, nil, PlanFromIssueInput{
		ID:   optionStringValue(options, "id", "_arg"),
		Slug: optionStringValue(options, "slug"),
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) planFromIssue(ctx context.Context, _ *mcp.CallToolRequest, input PlanFromIssueInput) (*mcp.CallToolResult, PlanFromIssueOutput, error) {
	identifier := issueRecordIdentifier(input.Slug, input.ID)
	if identifier == "" {
		return nil, PlanFromIssueOutput{}, core.E("planFromIssue", "issue slug or id is required", nil)
	}

	issueResult := s.handleIssueRecordGet(ctx, core.NewOptions(
		core.Option{Key: "id", Value: input.ID},
		core.Option{Key: "slug", Value: input.Slug},
	))
	if !issueResult.OK {
		if value, ok := issueResult.Value.(error); ok {
			return nil, PlanFromIssueOutput{}, value
		}
		return nil, PlanFromIssueOutput{}, core.E("planFromIssue", "failed to read issue", nil)
	}

	issueOutput, ok := issueResult.Value.(IssueOutput)
	if !ok {
		return nil, PlanFromIssueOutput{}, core.E("planFromIssue", "invalid issue output", nil)
	}
	if issueOutput.Issue.Title == "" {
		return nil, PlanFromIssueOutput{}, core.E("planFromIssue", "issue title is required", nil)
	}

	description := issueOutput.Issue.Description
	objective := description
	if objective == "" {
		objective = issueOutput.Issue.Title
	}
	tasks := planFromIssueTasks(description)

	phaseTitle := core.Concat("Resolve issue: ", issueOutput.Issue.Title)
	phaseCriteria := []string{"Issue is closed", "QA passes"}
	if len(issueOutput.Issue.Labels) > 0 {
		phaseCriteria = append(phaseCriteria, "Issue labels are preserved in the plan context")
	}

	_, createOutput, err := s.planCreate(ctx, nil, PlanCreateInput{
		Title:       issueOutput.Issue.Title,
		Slug:        planFromIssueSlug(issueOutput.Issue.Slug),
		Objective:   objective,
		Description: description,
		Context: map[string]any{
			"source_issue":          issueOutput.Issue,
			"source_issue_id":       issueOutput.Issue.ID,
			"source_issue_slug":     issueOutput.Issue.Slug,
			"source_issue_type":     issueOutput.Issue.Type,
			"source_issue_labels":   issueOutput.Issue.Labels,
			"source_issue_status":   issueOutput.Issue.Status,
			"source_issue_metadata": issueOutput.Issue.Metadata,
			"source_issue_state":    issueOutput.Issue.Status,
			"source_issue_meta":     issueOutput.Issue.Metadata,
		},
		Phases: []Phase{
			{
				Number:      1,
				Name:        phaseTitle,
				Description: description,
				Criteria:    phaseCriteria,
				Tasks:       tasks,
			},
		},
		Notes: core.Concat("Created from issue ", identifier),
	})
	if err != nil {
		return nil, PlanFromIssueOutput{}, err
	}

	planResult := readPlanResult(PlansRoot(), createOutput.ID)
	if !planResult.OK {
		err, _ := planResult.Value.(error)
		if err == nil {
			err = core.E("planFromIssue", "failed to read created plan", nil)
		}
		return nil, PlanFromIssueOutput{}, err
	}
	plan, ok := planResult.Value.(*Plan)
	if !ok || plan == nil {
		return nil, PlanFromIssueOutput{}, core.E("planFromIssue", "invalid created plan", nil)
	}

	return nil, PlanFromIssueOutput{
		Success: true,
		Issue:   issueOutput.Issue,
		Plan:    *plan,
		Path:    createOutput.Path,
	}, nil
}

func planFromIssueTasks(body string) []PlanTask {
	var tasks []PlanTask
	for _, line := range core.Split(body, "\n") {
		trimmed := core.Trim(line)
		if title, ok := planFromIssueTaskTitle(trimmed); ok {
			tasks = append(tasks, PlanTask{Title: title})
		}
	}
	return tasks
}

func planFromIssueTaskTitle(line string) (string, bool) {
	for _, prefix := range []string{"- [ ] ", "- [x] ", "- [X] ", "* [ ] ", "* [x] ", "* [X] "} {
		if core.HasPrefix(line, prefix) {
			title := core.Trim(core.TrimPrefix(line, prefix))
			if title != "" {
				return title, true
			}
			return "", false
		}
	}
	return "", false
}

// plan-from-issue fixtures should use issue slugs like `fix-auth`.
func planFromIssueSlug(slug string) string {
	if slug == "" {
		return ""
	}
	if core.HasPrefix(slug, "issue-") {
		return slug
	}
	return core.Concat("issue-", slug)
}
