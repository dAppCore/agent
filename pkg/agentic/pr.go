// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	forge_types "dappco.re/go/core/forge/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CreatePRInput is the input for agentic_create_pr.
//
//	input := agentic.CreatePRInput{Workspace: "core/go-io/task-42", Title: "Fix watcher panic"}
type CreatePRInput struct {
	Workspace string `json:"workspace"`         // workspace name (e.g. "core/go-io/task-42")
	Title     string `json:"title,omitempty"`   // PR title (default: task description)
	Body      string `json:"body,omitempty"`    // PR body (default: auto-generated)
	Base      string `json:"base,omitempty"`    // base branch (default: "main")
	DryRun    bool   `json:"dry_run,omitempty"` // preview without creating
}

// CreatePROutput is the output for agentic_create_pr.
//
//	out := agentic.CreatePROutput{Success: true, PRURL: "https://forge.example/core/go-io/pulls/12", PRNum: 12}
type CreatePROutput struct {
	Success bool   `json:"success"`
	PRURL   string `json:"pr_url,omitempty"`
	PRNum   int    `json:"pr_number,omitempty"`
	Title   string `json:"title"`
	Branch  string `json:"branch"`
	Repo    string `json:"repo"`
	Pushed  bool   `json:"pushed"`
}

func (s *PrepSubsystem) registerCreatePRTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_create_pr",
		Description: "Create a pull request from an agent workspace. Pushes the branch to Forge and opens a PR. Links to the source issue if one was tracked.",
	}, s.createPR)
}

func (s *PrepSubsystem) createPR(ctx context.Context, _ *mcp.CallToolRequest, input CreatePRInput) (*mcp.CallToolResult, CreatePROutput, error) {
	if input.Workspace == "" {
		return nil, CreatePROutput{}, core.E("createPR", "workspace is required", nil)
	}
	if s.forgeToken == "" {
		return nil, CreatePROutput{}, core.E("createPR", "no Forge token configured", nil)
	}

	workspaceDir := core.JoinPath(WorkspaceRoot(), input.Workspace)
	repoDir := WorkspaceRepoDir(workspaceDir)

	if !fs.IsDir(core.JoinPath(repoDir, ".git")) {
		return nil, CreatePROutput{}, core.E("createPR", core.Concat("workspace not found: ", input.Workspace), nil)
	}

	// Read workspace status for repo, branch, issue context
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		err, _ := result.Value.(error)
		return nil, CreatePROutput{}, core.E("createPR", "no status.json", err)
	}

	if workspaceStatus.Branch == "" {
		process := s.Core().Process()
		result := process.RunIn(ctx, repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
		if !result.OK {
			return nil, CreatePROutput{}, core.E("createPR", "failed to detect branch", nil)
		}
		workspaceStatus.Branch = core.Trim(result.Value.(string))
		if workspaceStatus.Branch == "" {
			return nil, CreatePROutput{}, core.E("createPR", "failed to detect branch", nil)
		}
	}

	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}
	base := input.Base
	if base == "" {
		base = "dev"
	}

	// Build PR title
	title := input.Title
	if title == "" {
		title = workspaceStatus.Task
	}
	if title == "" {
		title = core.Sprintf("Agent work on %s", workspaceStatus.Branch)
	}

	// Build PR body
	body := input.Body
	if body == "" {
		body = s.buildPRBody(workspaceStatus)
	}

	if input.DryRun {
		return nil, CreatePROutput{
			Success: true,
			Title:   title,
			Branch:  workspaceStatus.Branch,
			Repo:    workspaceStatus.Repo,
		}, nil
	}

	// Push branch to Forge (origin is the local clone, not Forge)
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, workspaceStatus.Repo)
	pushResult := s.Core().Process().RunIn(ctx, repoDir, "git", "push", forgeRemote, workspaceStatus.Branch)
	if !pushResult.OK {
		return nil, CreatePROutput{}, core.E("createPR", core.Concat("git push failed: ", pushResult.Value.(string)), nil)
	}

	// Create PR via Forge API
	pullRequestURL, pullRequestNumber, err := s.forgeCreatePR(ctx, org, workspaceStatus.Repo, workspaceStatus.Branch, base, title, body)
	if err != nil {
		return nil, CreatePROutput{}, core.E("createPR", "failed to create PR", err)
	}

	// Update status with PR URL
	workspaceStatus.PRURL = pullRequestURL
	writeStatusResult(workspaceDir, workspaceStatus)

	// Comment on issue if tracked
	if workspaceStatus.Issue > 0 {
		comment := core.Sprintf("Pull request created: %s", pullRequestURL)
		s.commentOnIssue(ctx, org, workspaceStatus.Repo, workspaceStatus.Issue, comment)
	}

	return nil, CreatePROutput{
		Success: true,
		PRURL:   pullRequestURL,
		PRNum:   pullRequestNumber,
		Title:   title,
		Branch:  workspaceStatus.Branch,
		Repo:    workspaceStatus.Repo,
		Pushed:  true,
	}, nil
}

func (s *PrepSubsystem) buildPRBody(workspaceStatus *WorkspaceStatus) string {
	b := core.NewBuilder()
	b.WriteString("## Summary\n\n")
	if workspaceStatus.Task != "" {
		b.WriteString(workspaceStatus.Task)
		b.WriteString("\n\n")
	}
	if workspaceStatus.Issue > 0 {
		b.WriteString(core.Sprintf("Closes #%d\n\n", workspaceStatus.Issue))
	}
	b.WriteString(core.Sprintf("**Agent:** %s\n", workspaceStatus.Agent))
	b.WriteString(core.Sprintf("**Runs:** %d\n", workspaceStatus.Runs))
	b.WriteString("\n---\n*Created by agentic dispatch*\n")
	return b.String()
}

func (s *PrepSubsystem) forgeCreatePR(ctx context.Context, org, repo, head, base, title, body string) (string, int, error) {
	var pullRequest pullRequestView
	err := s.forge.Client().Post(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls", org, repo), &forge_types.CreatePullRequestOption{
		Title: title,
		Body:  body,
		Head:  head,
		Base:  base,
	}, &pullRequest)
	if err != nil {
		return "", 0, core.E("forgeCreatePR", "create PR failed", err)
	}
	return pullRequest.HTMLURL, int(pullRequestNumber(pullRequest)), nil
}

func (s *PrepSubsystem) commentOnIssue(ctx context.Context, org, repo string, issue int, comment string) {
	s.forge.Issues.CreateComment(ctx, org, repo, int64(issue), comment)
}

// ListPRsInput is the input for agentic_list_prs.
//
//	input := agentic.ListPRsInput{Org: "core", Repo: "go-io", State: "open", Limit: 10}
type ListPRsInput struct {
	Org   string `json:"org,omitempty"`   // forge org (default "core")
	Repo  string `json:"repo,omitempty"`  // specific repo, or empty for all
	State string `json:"state,omitempty"` // "open" (default), "closed", "all"
	Limit int    `json:"limit,omitempty"` // max results (default 20)
}

// ListPRsOutput is the output for agentic_list_prs.
//
//	out := agentic.ListPRsOutput{Success: true, Count: 2, PRs: []agentic.PRInfo{{Repo: "go-io", Number: 12}}}
type ListPRsOutput struct {
	Success bool     `json:"success"`
	Count   int      `json:"count"`
	PRs     []PRInfo `json:"prs"`
}

// PRInfo represents a pull request.
//
//	pr := agentic.PRInfo{Repo: "go-io", Number: 12, Title: "Migrate pkg/fs", Branch: "agent/migrate-fs"}
type PRInfo struct {
	Repo      string   `json:"repo"`
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Author    string   `json:"author"`
	Branch    string   `json:"branch"`
	Base      string   `json:"base"`
	Labels    []string `json:"labels,omitempty"`
	Mergeable bool     `json:"mergeable"`
	URL       string   `json:"url"`
}

func (s *PrepSubsystem) registerListPRsTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_list_prs",
		Description: "List pull requests across Forge repos. Filter by org, repo, and state (open/closed/all).",
	}, s.listPRs)
}

func (s *PrepSubsystem) listPRs(ctx context.Context, _ *mcp.CallToolRequest, input ListPRsInput) (*mcp.CallToolResult, ListPRsOutput, error) {
	if s.forgeToken == "" {
		return nil, ListPRsOutput{}, core.E("listPRs", "no Forge token configured", nil)
	}

	if input.Org == "" {
		input.Org = "core"
	}
	if input.State == "" {
		input.State = "open"
	}
	if input.Limit == 0 {
		input.Limit = 20
	}

	var repos []string
	if input.Repo != "" {
		repos = []string{input.Repo}
	} else {
		var err error
		repos, err = s.listOrgRepos(ctx, input.Org)
		if err != nil {
			return nil, ListPRsOutput{}, err
		}
	}

	var allPRs []PRInfo

	for _, repo := range repos {
		prs, err := s.listRepoPRs(ctx, input.Org, repo, input.State)
		if err != nil {
			continue
		}
		allPRs = append(allPRs, prs...)

		if len(allPRs) >= input.Limit {
			break
		}
	}

	if len(allPRs) > input.Limit {
		allPRs = allPRs[:input.Limit]
	}

	return nil, ListPRsOutput{
		Success: true,
		Count:   len(allPRs),
		PRs:     allPRs,
	}, nil
}

func (s *PrepSubsystem) listRepoPRs(ctx context.Context, org, repo, state string) ([]PRInfo, error) {
	var pullRequests []pullRequestView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls?limit=50&page=1", org, repo), &pullRequests)
	if err != nil {
		return nil, core.E("listRepoPRs", core.Concat("failed to list PRs for ", repo), err)
	}

	var result []PRInfo
	for _, pullRequest := range pullRequests {
		pullRequestState := pullRequest.State
		if pullRequestState == "" {
			pullRequestState = "open"
		}
		if state != "" && state != "all" && pullRequestState != state {
			continue
		}
		var labels []string
		for _, label := range pullRequest.Labels {
			labels = append(labels, label.Name)
		}
		result = append(result, PRInfo{
			Repo:      repo,
			Number:    int(pullRequestNumber(pullRequest)),
			Title:     pullRequest.Title,
			State:     pullRequestState,
			Author:    pullRequestAuthor(pullRequest),
			Branch:    pullRequest.Head.Ref,
			Base:      pullRequest.Base.Ref,
			Labels:    labels,
			Mergeable: pullRequest.Mergeable,
			URL:       pullRequest.HTMLURL,
		})
	}

	return result, nil
}
