// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EpicInput struct {
	Repo     string   `json:"repo"`
	Org      string   `json:"org,omitempty"`
	Title    string   `json:"title"`
	Body     string   `json:"body,omitempty"`
	Tasks    []string `json:"tasks"`
	Labels   []string `json:"labels,omitempty"`
	Dispatch bool     `json:"dispatch,omitempty"`
	Agent    string   `json:"agent,omitempty"`
	Template string   `json:"template,omitempty"`
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

func (s *PrepSubsystem) registerEpicTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
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

	labelIDs := s.resolveLabelIDs(ctx, input.Org, input.Repo, labels)

	var children []ChildRef
	for _, task := range input.Tasks {
		child, err := s.createIssue(ctx, input.Org, input.Repo, task, "", labelIDs)
		if err != nil {
			continue
		}
		children = append(children, child)
	}

	body := core.NewBuilder()
	if input.Body != "" {
		body.WriteString(input.Body)
		body.WriteString("\n\n")
	}
	body.WriteString("## Tasks\n\n")
	for _, child := range children {
		body.WriteString(core.Sprintf("- [ ] #%d %s\n", child.Number, child.Title))
	}

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
