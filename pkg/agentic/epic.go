// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EpicInput struct {
	Repo     string   `json:"repo"`               // Target repo (e.g. "go-scm")
	Org      string   `json:"org,omitempty"`      // Forge org (default "core")
	Title    string   `json:"title"`              // Epic title
	Body     string   `json:"body,omitempty"`     // Epic description (above checklist)
	Tasks    []string `json:"tasks"`              // Sub-task titles (become child issues)
	Labels   []string `json:"labels,omitempty"`   // Labels for epic + children (e.g. ["agentic"])
	Dispatch bool     `json:"dispatch,omitempty"` // Auto-dispatch agents to each child
	Agent    string   `json:"agent,omitempty"`    // Agent type for dispatch (default "claude")
	Template string   `json:"template,omitempty"` // Prompt template for dispatch (default "coding")
}

type EpicOutput struct {
	Success    bool       `json:"success"`
	EpicNumber int        `json:"epic_number"`
	EpicURL    string     `json:"epic_url"`
	Children   []ChildRef `json:"children"`
	Dispatched int        `json:"dispatched,omitempty"`
}

type ChildRef struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

func (s *PrepSubsystem) registerEpicTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_create_epic",
		Description: "Create an epic issue with child issues on Forge. Each task becomes a child issue linked via checklist. Optionally auto-dispatch agents to work each child.",
	}, s.createEpic)
}

func (s *PrepSubsystem) createEpic(ctx context.Context, callRequest *mcp.CallToolRequest, input EpicInput) (*mcp.CallToolResult, EpicOutput, error) {
	if input.Title == "" {
		return nil, EpicOutput{}, core.E("createEpic", "title is required", nil)
	}
	if len(input.Tasks) == 0 {
		return nil, EpicOutput{}, core.E("createEpic", "at least one task is required", nil)
	}
	if s.forgeToken == "" {
		return nil, EpicOutput{}, core.E("createEpic", "no Forge token configured", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "claude"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	// Ensure "agentic" label exists
	labels := input.Labels
	hasAgentic := false
	for _, l := range labels {
		if l == "agentic" {
			hasAgentic = true
			break
		}
	}
	if !hasAgentic {
		labels = append(labels, "agentic")
	}

	// Get label IDs
	labelIDs := s.resolveLabelIDs(ctx, input.Org, input.Repo, labels)

	// Step 1: Create child issues first (we need their numbers for the checklist)
	var children []ChildRef
	for _, task := range input.Tasks {
		child, err := s.createIssue(ctx, input.Org, input.Repo, task, "", labelIDs)
		if err != nil {
			continue // Skip failed children, create what we can
		}
		children = append(children, child)
	}

	// Step 2: Build epic body with checklist
	body := core.NewBuilder()
	if input.Body != "" {
		body.WriteString(input.Body)
		body.WriteString("\n\n")
	}
	body.WriteString("## Tasks\n\n")
	for _, child := range children {
		body.WriteString(core.Sprintf("- [ ] #%d %s\n", child.Number, child.Title))
	}

	// Step 3: Create epic issue
	epicLabels := append(labelIDs, s.resolveLabelIDs(ctx, input.Org, input.Repo, []string{"epic"})...)
	epic, err := s.createIssue(ctx, input.Org, input.Repo, input.Title, body.String(), epicLabels)
	if err != nil {
		return nil, EpicOutput{}, core.E("createEpic", "failed to create epic", err)
	}

	out := EpicOutput{
		Success:    true,
		EpicNumber: epic.Number,
		EpicURL:    epic.URL,
		Children:   children,
	}

	// Step 4: Optionally dispatch agents to each child
	if input.Dispatch {
		for _, child := range children {
			_, _, err := s.dispatch(ctx, callRequest, DispatchInput{
				Repo:     input.Repo,
				Org:      input.Org,
				Task:     child.Title,
				Agent:    input.Agent,
				Template: input.Template,
				Issue:    child.Number,
			})
			if err == nil {
				out.Dispatched++
			}
		}
	}

	return nil, out, nil
}

// child, err := s.createIssue(ctx, "core", "go-scm", "Port agentic plans", "", nil)
func (s *PrepSubsystem) createIssue(ctx context.Context, org, repo, title, body string, labelIDs []int64) (ChildRef, error) {
	payload := map[string]any{
		"title": title,
	}
	if body != "" {
		payload["body"] = body
	}
	if len(labelIDs) > 0 {
		payload["labels"] = labelIDs
	}

	data := core.JSONMarshalString(payload)
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues", s.forgeURL, org, repo)
	httpResult := HTTPPost(ctx, url, data, s.forgeToken, "token")
	if !httpResult.OK {
		return ChildRef{}, core.E("createIssue", "create issue request failed", nil)
	}

	var createdIssue struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
	}
	core.JSONUnmarshalString(httpResult.Value.(string), &createdIssue)

	return ChildRef{
		Number: createdIssue.Number,
		Title:  title,
		URL:    createdIssue.HTMLURL,
	}, nil
}

// labelIDs := s.resolveLabelIDs(ctx, "core", "go-scm", []string{"agentic", "epic"})
func (s *PrepSubsystem) resolveLabelIDs(ctx context.Context, org, repo string, names []string) []int64 {
	if len(names) == 0 {
		return nil
	}

	// Fetch existing labels
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels?limit=50", s.forgeURL, org, repo)
	httpResult := HTTPGet(ctx, url, s.forgeToken, "token")
	if !httpResult.OK {
		return nil
	}

	var existing []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	core.JSONUnmarshalString(httpResult.Value.(string), &existing)

	nameToID := make(map[string]int64)
	for _, l := range existing {
		nameToID[l.Name] = l.ID
	}

	var ids []int64
	for _, name := range names {
		if id, ok := nameToID[name]; ok {
			ids = append(ids, id)
		} else {
			// Create the label
			id := s.createLabel(ctx, org, repo, name)
			if id > 0 {
				ids = append(ids, id)
			}
		}
	}

	return ids
}

// id := s.createLabel(ctx, "core", "go-scm", "agentic")
func (s *PrepSubsystem) createLabel(ctx context.Context, org, repo, name string) int64 {
	colours := map[string]string{
		"agentic":     "#7c3aed",
		"epic":        "#dc2626",
		"bug":         "#ef4444",
		"help-wanted": "#22c55e",
	}
	colour := colours[name]
	if colour == "" {
		colour = "#6b7280"
	}

	payload := core.JSONMarshalString(map[string]string{
		"name":  name,
		"color": colour,
	})

	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	httpResult := HTTPPost(ctx, url, payload, s.forgeToken, "token")
	if !httpResult.OK {
		return 0
	}

	var createdLabel struct {
		ID int64 `json:"id"`
	}
	core.JSONUnmarshalString(httpResult.Value.(string), &createdLabel)
	return createdLabel.ID
}
