// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	forge_types "dappco.re/go/core/forge/types"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.CreatePRInput{Workspace: "core/go-io/task-42", Title: "Fix watcher panic"}
type CreatePRInput struct {
	Workspace string `json:"workspace"`
	Title     string `json:"title,omitempty"`
	Body      string `json:"body,omitempty"`
	Base      string `json:"base,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
}

// out := agentic.CreatePROutput{Success: true, PRURL: "https://forge.example/core/go-io/pulls/12", PRNum: 12}
type CreatePROutput struct {
	Success bool   `json:"success"`
	PRURL   string `json:"pr_url,omitempty"`
	PRNum   int    `json:"pr_number,omitempty"`
	Title   string `json:"title"`
	Branch  string `json:"branch"`
	Repo    string `json:"repo"`
	Pushed  bool   `json:"pushed"`
}

func (s *PrepSubsystem) registerCreatePRTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_create_pr",
		Description: "Create a pull request from an agent workspace. Pushes the branch to Forge and opens a PR. Links to the source issue if one was tracked.",
	}, s.createPR)
}

// input := agentic.PRGetInput{Org: "core", Repo: "go-io", Number: 42}
type PRGetInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

// out := agentic.PRGetOutput{Success: true, PR: agentic.PRInfo{Repo: "go-io", Number: 42}}
type PRGetOutput struct {
	Success bool   `json:"success"`
	PR      PRInfo `json:"pr"`
}

// input := agentic.PRMergeInput{Org: "core", Repo: "go-io", Number: 42, Method: "squash"}
type PRMergeInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Method string `json:"method,omitempty"`
}

// out := agentic.PRMergeOutput{Success: true, Repo: "go-io", Number: 42, State: "merged"}
type PRMergeOutput struct {
	Success bool   `json:"success"`
	Org     string `json:"org,omitempty"`
	Repo    string `json:"repo"`
	Number  int    `json:"number"`
	Method  string `json:"method,omitempty"`
	State   string `json:"state,omitempty"`
	PR      PRInfo `json:"pr,omitempty"`
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

	title := input.Title
	if title == "" {
		title = workspaceStatus.Task
	}
	if title == "" {
		title = core.Sprintf("Agent work on %s", workspaceStatus.Branch)
	}

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

	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, workspaceStatus.Repo)
	pushResult := s.Core().Process().RunIn(ctx, repoDir, "git", "push", forgeRemote, workspaceStatus.Branch)
	if !pushResult.OK {
		return nil, CreatePROutput{}, core.E("createPR", core.Concat("git push failed: ", pushResult.Value.(string)), nil)
	}

	pullRequestURL, pullRequestNumber, err := s.forgeCreatePR(ctx, org, workspaceStatus.Repo, workspaceStatus.Branch, base, title, body)
	if err != nil {
		return nil, CreatePROutput{}, core.E("createPR", "failed to create PR", err)
	}

	workspaceStatus.PRURL = pullRequestURL
	writeStatusResult(workspaceDir, workspaceStatus)
	s.cleanupForgeBranch(ctx, repoDir, forgeRemote, workspaceStatus.Branch)

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

func (s *PrepSubsystem) registerPRTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_pr_get",
		Description: "Read a pull request from Forge by repository and pull request number.",
	}, s.prGet)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "pr_get",
		Description: "Read a pull request from Forge by repository and pull request number.",
	}, s.prGet)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_pr_list",
		Description: "List pull requests across Forge repos. Filter by org, repo, and state.",
	}, s.prList)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "pr_list",
		Description: "List pull requests across Forge repos. Filter by org, repo, and state.",
	}, s.prList)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_pr_merge",
		Description: "Merge a pull request on Forge by repository and pull request number.",
	}, s.prMerge)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "pr_merge",
		Description: "Merge a pull request on Forge by repository and pull request number.",
	}, s.prMerge)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_pr_close",
		Description: "Close a pull request on Forge by repository and pull request number.",
	}, s.closePR)

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "pr_close",
		Description: "Close a pull request on Forge by repository and pull request number.",
	}, s.closePR)
}

func (s *PrepSubsystem) prGet(ctx context.Context, _ *mcp.CallToolRequest, input PRGetInput) (*mcp.CallToolResult, PRGetOutput, error) {
	if s.forgeToken == "" {
		return nil, PRGetOutput{}, core.E("prGet", "no Forge token configured", nil)
	}
	if input.Repo == "" || input.Number <= 0 {
		return nil, PRGetOutput{}, core.E("prGet", "repo and number are required", nil)
	}

	org := input.Org
	if org == "" {
		org = "core"
	}

	var pr pullRequestView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d", org, input.Repo, input.Number), &pr)
	if err != nil {
		return nil, PRGetOutput{}, core.E("prGet", core.Concat("failed to read PR ", core.Sprint(input.Number)), err)
	}

	return nil, PRGetOutput{
		Success: true,
		PR: PRInfo{
			Repo:      input.Repo,
			Number:    int(pullRequestNumber(pr)),
			Title:     pr.Title,
			State:     pr.State,
			Author:    pullRequestAuthor(pr),
			Branch:    pr.Head.Ref,
			Base:      pr.Base.Ref,
			Mergeable: pr.Mergeable,
			URL:       pr.HTMLURL,
		},
	}, nil
}

func (s *PrepSubsystem) prList(ctx context.Context, _ *mcp.CallToolRequest, input ListPRsInput) (*mcp.CallToolResult, ListPRsOutput, error) {
	return s.listPRs(ctx, nil, input)
}

func (s *PrepSubsystem) prMerge(ctx context.Context, _ *mcp.CallToolRequest, input PRMergeInput) (*mcp.CallToolResult, PRMergeOutput, error) {
	if s.forgeToken == "" {
		return nil, PRMergeOutput{}, core.E("prMerge", "no Forge token configured", nil)
	}
	if input.Repo == "" || input.Number <= 0 {
		return nil, PRMergeOutput{}, core.E("prMerge", "repo and number are required", nil)
	}

	org := input.Org
	if org == "" {
		org = "core"
	}
	method := input.Method
	if method == "" {
		method = "merge"
	}

	if err := s.forge.Pulls.Merge(ctx, org, input.Repo, int64(input.Number), method); err != nil {
		return nil, PRMergeOutput{}, core.E("prMerge", core.Concat("failed to merge PR ", core.Sprint(input.Number)), err)
	}

	output := PRMergeOutput{
		Success: true,
		Org:     org,
		Repo:    input.Repo,
		Number:  input.Number,
		Method:  method,
		State:   "merged",
	}

	if _, prOutput, err := s.prGet(ctx, nil, PRGetInput{Org: org, Repo: input.Repo, Number: input.Number}); err == nil {
		output.PR = prOutput.PR
	}

	return nil, output, nil
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

// input := agentic.ListPRsInput{Org: "core", Repo: "go-io", State: "open", Limit: 10}
type ListPRsInput struct {
	Org   string `json:"org,omitempty"`
	Repo  string `json:"repo,omitempty"`
	State string `json:"state,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

// out := agentic.ListPRsOutput{Success: true, Count: 2, PRs: []agentic.PRInfo{{Repo: "go-io", Number: 12}}}
type ListPRsOutput struct {
	Success bool     `json:"success"`
	Count   int      `json:"count"`
	PRs     []PRInfo `json:"prs"`
}

// input := agentic.ClosePRInput{Org: "core", Repo: "go-io", Number: 12}
type ClosePRInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

// out := agentic.ClosePROutput{Success: true, Repo: "go-io", Number: 12, State: "closed"}
type ClosePROutput struct {
	Success bool   `json:"success"`
	Org     string `json:"org,omitempty"`
	Repo    string `json:"repo"`
	Number  int    `json:"number"`
	State   string `json:"state,omitempty"`
}

// pr := agentic.PRInfo{Repo: "go-io", Number: 12, Title: "Migrate pkg/fs", Branch: "agent/migrate-fs"}
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

func (s *PrepSubsystem) registerListPRsTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_list_prs",
		Description: "List pull requests across Forge repos. Filter by org, repo, and state (open/closed/all).",
	}, s.listPRs)
}

func (s *PrepSubsystem) registerClosePRTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_close_pr",
		Description: "Close a pull request on Forge by repository and pull request number.",
	}, s.closePR)
}

// s.registerDeleteBranchTool(svc)
func (s *PrepSubsystem) registerDeleteBranchTool(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "agentic_delete_branch",
		Description: "Delete a branch on the Forge remote. Used after successful merge or close to clean up agent branches.",
	}, s.deleteBranch)
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

	var repositories []string
	if input.Repo != "" {
		repositories = []string{input.Repo}
	} else {
		var repositoryErr error
		repositories, repositoryErr = s.listOrgRepos(ctx, input.Org)
		if repositoryErr != nil {
			return nil, ListPRsOutput{}, repositoryErr
		}
	}

	var allPullRequests []PRInfo

	for _, repo := range repositories {
		prs, err := s.listRepoPRs(ctx, input.Org, repo, input.State)
		if err != nil {
			continue
		}
		allPullRequests = append(allPullRequests, prs...)

		if len(allPullRequests) >= input.Limit {
			break
		}
	}

	if len(allPullRequests) > input.Limit {
		allPullRequests = allPullRequests[:input.Limit]
	}

	return nil, ListPRsOutput{
		Success: true,
		Count:   len(allPullRequests),
		PRs:     allPullRequests,
	}, nil
}

func (s *PrepSubsystem) closePR(ctx context.Context, _ *mcp.CallToolRequest, input ClosePRInput) (*mcp.CallToolResult, ClosePROutput, error) {
	if s.forgeToken == "" {
		return nil, ClosePROutput{}, core.E("closePR", "no Forge token configured", nil)
	}
	if s.forge == nil {
		return nil, ClosePROutput{}, core.E("closePR", "forge client is not configured", nil)
	}
	if input.Repo == "" || input.Number <= 0 {
		return nil, ClosePROutput{}, core.E("closePR", "repo and number are required", nil)
	}

	org := input.Org
	if org == "" {
		org = "core"
	}

	var pr pullRequestView
	err := s.forge.Client().Patch(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d", org, input.Repo, input.Number), &forge_types.EditPullRequestOption{
		State: "closed",
	}, &pr)
	if err != nil {
		return nil, ClosePROutput{}, core.E("closePR", core.Concat("failed to close PR ", core.Sprint(input.Number)), err)
	}

	state := pr.State
	if state == "" {
		state = "closed"
	}

	return nil, ClosePROutput{
		Success: true,
		Org:     org,
		Repo:    input.Repo,
		Number:  input.Number,
		State:   state,
	}, nil
}

// input := agentic.DeleteBranchInput{Org: "core", Repo: "go-io", Branch: "agent/fix-tests"}
type DeleteBranchInput struct {
	// input := agentic.DeleteBranchInput{Org: "core"}
	Org string `json:"org,omitempty"`
	// input := agentic.DeleteBranchInput{Repo: "go-io"}
	Repo string `json:"repo"`
	// input := agentic.DeleteBranchInput{Branch: "agent/fix-tests"}
	Branch string `json:"branch"`
}

// out := agentic.DeleteBranchOutput{Success: true, Repo: "go-io", Branch: "agent/fix-tests"}
type DeleteBranchOutput struct {
	// out := agentic.DeleteBranchOutput{Success: true}
	Success bool `json:"success"`
	// out := agentic.DeleteBranchOutput{Org: "core"}
	Org string `json:"org,omitempty"`
	// out := agentic.DeleteBranchOutput{Repo: "go-io"}
	Repo string `json:"repo"`
	// out := agentic.DeleteBranchOutput{Branch: "agent/fix-tests"}
	Branch string `json:"branch"`
}

// s.deleteBranch(ctx, nil, agentic.DeleteBranchInput{Repo: "go-io", Branch: "agent/fix-tests"})
func (s *PrepSubsystem) deleteBranch(ctx context.Context, _ *mcp.CallToolRequest, input DeleteBranchInput) (*mcp.CallToolResult, DeleteBranchOutput, error) {
	if s.forgeToken == "" {
		return nil, DeleteBranchOutput{}, core.E("deleteBranch", "no Forge token configured", nil)
	}
	if s.forge == nil {
		return nil, DeleteBranchOutput{}, core.E("deleteBranch", "forge client is not configured", nil)
	}
	if input.Repo == "" || input.Branch == "" {
		return nil, DeleteBranchOutput{}, core.E("deleteBranch", "repo and branch are required", nil)
	}

	org := input.Org
	if org == "" {
		org = "core"
	}

	if err := s.forge.Branches.DeleteBranch(ctx, org, input.Repo, input.Branch); err != nil {
		return nil, DeleteBranchOutput{}, core.E("deleteBranch", core.Concat("failed to delete branch ", input.Branch), err)
	}

	return nil, DeleteBranchOutput{
		Success: true,
		Org:     org,
		Repo:    input.Repo,
		Branch:  input.Branch,
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
