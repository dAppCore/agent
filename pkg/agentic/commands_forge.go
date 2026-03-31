// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"strconv"

	core "dappco.re/go/core"
	"dappco.re/go/core/forge"
	forge_types "dappco.re/go/core/forge/types"
)

type issueView struct {
	Index   int64  `json:"index"`
	Number  int64  `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

type prBranchView struct {
	Ref string `json:"ref"`
}

type prUserView struct {
	Login    string `json:"login"`
	UserName string `json:"username"`
}

type prLabelView struct {
	Name string `json:"name"`
}

type pullRequestView struct {
	Index     int64         `json:"index"`
	Number    int64         `json:"number"`
	Title     string        `json:"title"`
	State     string        `json:"state"`
	Mergeable bool          `json:"mergeable"`
	HTMLURL   string        `json:"html_url"`
	Body      string        `json:"body"`
	Head      prBranchView  `json:"head"`
	Base      prBranchView  `json:"base"`
	User      *prUserView   `json:"user"`
	Labels    []prLabelView `json:"labels"`
}

func issueNumber(issue issueView) int64 {
	if issue.Index != 0 {
		return issue.Index
	}
	return issue.Number
}

func pullRequestNumber(pr pullRequestView) int64 {
	if pr.Index != 0 {
		return pr.Index
	}
	return pr.Number
}

func pullRequestAuthor(pr pullRequestView) string {
	if pr.User == nil {
		return ""
	}
	if pr.User.UserName != "" {
		return pr.User.UserName
	}
	return pr.User.Login
}

// org, repo, num := parseForgeArgs(core.NewOptions(
//
//	core.Option{Key: "org", Value: "core"},
//	core.Option{Key: "_arg", Value: "go-io"},
//	core.Option{Key: "number", Value: "42"},
//
// ))
func parseForgeArgs(options core.Options) (org, repo string, num int64) {
	org = options.String("org")
	if org == "" {
		org = "core"
	}
	repo = options.String("_arg")
	if v := options.String("number"); v != "" {
		num, _ = strconv.ParseInt(v, 10, 64)
	}
	return
}

func fmtIndex(n int64) string { return strconv.FormatInt(n, 10) }

// c.Command("issue/get", core.Command{Description: "Get a Forge issue", Action: s.cmdIssueGet})
// c.Command("pr/merge", core.Command{Description: "Merge a Forge PR", Action: s.cmdPRMerge})
func (s *PrepSubsystem) registerForgeCommands() {
	c := s.Core()
	c.Command("issue/get", core.Command{Description: "Get a Forge issue", Action: s.cmdIssueGet})
	c.Command("issue/list", core.Command{Description: "List Forge issues for a repo", Action: s.cmdIssueList})
	c.Command("issue/comment", core.Command{Description: "Comment on a Forge issue", Action: s.cmdIssueComment})
	c.Command("issue/create", core.Command{Description: "Create a Forge issue", Action: s.cmdIssueCreate})
	c.Command("pr/get", core.Command{Description: "Get a Forge PR", Action: s.cmdPRGet})
	c.Command("pr/list", core.Command{Description: "List Forge PRs for a repo", Action: s.cmdPRList})
	c.Command("pr/merge", core.Command{Description: "Merge a Forge PR", Action: s.cmdPRMerge})
	c.Command("pr/close", core.Command{Description: "Close a Forge PR", Action: s.cmdPRClose})
	c.Command("repo/get", core.Command{Description: "Get Forge repo info", Action: s.cmdRepoGet})
	c.Command("repo/list", core.Command{Description: "List Forge repos for an org", Action: s.cmdRepoList})
}

func (s *PrepSubsystem) cmdIssueGet(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, num := parseForgeArgs(options)
	if repo == "" || num == 0 {
		core.Print(nil, "usage: core-agent issue get <repo> --number=N [--org=core]")
		return core.Result{Value: core.E("agentic.cmdIssueGet", "repo and number are required", nil), OK: false}
	}
	var issue issueView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues/%d", org, repo, num), &issue)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "#%d %s", issueNumber(issue), issue.Title)
	core.Print(nil, "  state:  %s", issue.State)
	core.Print(nil, "  url:    %s", issue.HTMLURL)
	if issue.Body != "" {
		core.Print(nil, "")
		core.Print(nil, "%s", issue.Body)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdIssueList(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, _ := parseForgeArgs(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent issue list <repo> [--org=core]")
		return core.Result{Value: core.E("agentic.cmdIssueList", "repo is required", nil), OK: false}
	}
	var issues []issueView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues?limit=50&page=1", org, repo), &issues)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	for _, issue := range issues {
		core.Print(nil, "  #%-4d %-6s %s", issueNumber(issue), issue.State, issue.Title)
	}
	if len(issues) == 0 {
		core.Print(nil, "  no issues")
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdIssueComment(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, num := parseForgeArgs(options)
	body := options.String("body")
	if repo == "" || num == 0 || body == "" {
		core.Print(nil, "usage: core-agent issue comment <repo> --number=N --body=\"text\" [--org=core]")
		return core.Result{Value: core.E("agentic.cmdIssueComment", "repo, number, and body are required", nil), OK: false}
	}
	comment, err := s.forge.Issues.CreateComment(ctx, org, repo, num, body)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "comment #%d created on %s/%s#%d", comment.ID, org, repo, num)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdIssueCreate(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, _ := parseForgeArgs(options)
	title := options.String("title")
	body := options.String("body")
	labels := options.String("labels")
	milestone := options.String("milestone")
	assignee := options.String("assignee")
	ref := options.String("ref")
	if repo == "" || title == "" {
		core.Print(nil, "usage: core-agent issue create <repo> --title=\"...\" [--body=\"...\"] [--labels=\"agentic,bug\"] [--milestone=\"v0.2.0\"] [--assignee=virgil] [--ref=dev] [--org=core]")
		return core.Result{Value: core.E("agentic.cmdIssueCreate", "repo and title are required", nil), OK: false}
	}

	createOptions := &forge_types.CreateIssueOption{Title: title, Body: body, Ref: ref}

	if milestone != "" {
		var milestones []forge_types.Milestone
		err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/milestones", org, repo), &milestones)
		if err == nil {
			for _, m := range milestones {
				if m.Title == milestone {
					createOptions.Milestone = m.ID
					break
				}
			}
		}
	}
	if assignee != "" {
		createOptions.Assignees = []string{assignee}
	}
	if labels != "" {
		labelNames := core.Split(labels, ",")
		allLabels, err := s.forge.Labels.ListRepoLabels(ctx, org, repo)
		if err == nil {
			for _, name := range labelNames {
				name = core.Trim(name)
				for _, l := range allLabels {
					if l.Name == name {
						createOptions.Labels = append(createOptions.Labels, l.ID)
						break
					}
				}
			}
		}
	}

	issue, err := s.forge.Issues.Create(ctx, forge.Params{"owner": org, "repo": repo}, createOptions)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "#%d %s", issue.Index, issue.Title)
	core.Print(nil, "  url: %s", issue.HTMLURL)
	return core.Result{Value: issue.Index, OK: true}
}

func (s *PrepSubsystem) cmdPRGet(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, num := parseForgeArgs(options)
	if repo == "" || num == 0 {
		core.Print(nil, "usage: core-agent pr get <repo> --number=N [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPRGet", "repo and number are required", nil), OK: false}
	}
	var pr pullRequestView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d", org, repo, num), &pr)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "#%d %s", pullRequestNumber(pr), pr.Title)
	core.Print(nil, "  state:     %s", pr.State)
	core.Print(nil, "  head:      %s", pr.Head.Ref)
	core.Print(nil, "  base:      %s", pr.Base.Ref)
	core.Print(nil, "  mergeable: %v", pr.Mergeable)
	core.Print(nil, "  url:       %s", pr.HTMLURL)
	if pr.Body != "" {
		core.Print(nil, "")
		core.Print(nil, "%s", pr.Body)
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPRList(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, _ := parseForgeArgs(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent pr list <repo> [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPRList", "repo is required", nil), OK: false}
	}
	var prs []pullRequestView
	err := s.forge.Client().Get(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls?limit=50&page=1", org, repo), &prs)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	for _, pr := range prs {
		core.Print(nil, "  #%-4d %-6s %s → %s  %s", pullRequestNumber(pr), pr.State, pr.Head.Ref, pr.Base.Ref, pr.Title)
	}
	if len(prs) == 0 {
		core.Print(nil, "  no PRs")
	}
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPRMerge(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, num := parseForgeArgs(options)
	method := options.String("method")
	if method == "" {
		method = "merge"
	}
	if repo == "" || num == 0 {
		core.Print(nil, "usage: core-agent pr merge <repo> --number=N [--method=merge|rebase|squash] [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPRMerge", "repo and number are required", nil), OK: false}
	}
	if err := s.forge.Pulls.Merge(ctx, org, repo, num, method); err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "merged %s/%s#%d via %s", org, repo, num, method)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdPRClose(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, num := parseForgeArgs(options)
	if repo == "" || num == 0 {
		core.Print(nil, "usage: core-agent pr close <repo> --number=N [--org=core]")
		return core.Result{Value: core.E("agentic.cmdPRClose", "repo and number are required", nil), OK: false}
	}

	var pr pullRequestView
	err := s.forge.Client().Patch(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d", org, repo, num), &forge_types.EditPullRequestOption{
		State: "closed",
	}, &pr)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "closed %s/%s#%d", org, repo, num)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdRepoGet(options core.Options) core.Result {
	ctx := context.Background()
	org, repo, _ := parseForgeArgs(options)
	if repo == "" {
		core.Print(nil, "usage: core-agent repo get <repo> [--org=core]")
		return core.Result{Value: core.E("agentic.cmdRepoGet", "repo is required", nil), OK: false}
	}
	repositoryResult, err := s.forge.Repos.Get(ctx, forge.Params{"owner": org, "repo": repo})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	core.Print(nil, "%s/%s", repositoryResult.Owner.UserName, repositoryResult.Name)
	core.Print(nil, "  description: %s", repositoryResult.Description)
	core.Print(nil, "  default:     %s", repositoryResult.DefaultBranch)
	core.Print(nil, "  private:     %v", repositoryResult.Private)
	core.Print(nil, "  archived:    %v", repositoryResult.Archived)
	core.Print(nil, "  url:         %s", repositoryResult.HTMLURL)
	return core.Result{OK: true}
}

func (s *PrepSubsystem) cmdRepoList(options core.Options) core.Result {
	ctx := context.Background()
	org := options.String("org")
	if org == "" {
		org = "core"
	}
	repos, err := s.forge.Repos.ListOrgRepos(ctx, org)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}
	for _, repository := range repos {
		archived := ""
		if repository.Archived {
			archived = " (archived)"
		}
		core.Print(nil, "  %-30s %s%s", repository.Name, repository.Description, archived)
	}
	core.Print(nil, "\n  %d repos", len(repos))
	return core.Result{OK: true}
}
