// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"strconv"

	"dappco.re/go/core"
	"dappco.re/go/core/forge"
)

// newForgeClient creates a Forge client from env config.
func newForgeClient() *forge.Forge {
	url := core.Env("FORGE_URL")
	if url == "" {
		url = "https://forge.lthn.ai"
	}
	token := core.Env("FORGE_TOKEN")
	if token == "" {
		token = core.Env("GITEA_TOKEN")
	}
	return forge.NewForge(url, token)
}

// parseArgs extracts org and repo from opts. First positional arg is repo, --org flag defaults to "core".
func parseArgs(opts core.Options) (org, repo string, num int64) {
	org = opts.String("org")
	if org == "" {
		org = "core"
	}
	repo = opts.String("_arg")
	if v := opts.String("number"); v != "" {
		num, _ = strconv.ParseInt(v, 10, 64)
	}
	return
}

func fmtIndex(n int64) string { return strconv.FormatInt(n, 10) }

func registerForgeCommands(c *core.Core) {
	ctx := context.Background()

	// --- Issues ---

	c.Command("issue/get", core.Command{
		Description: "Get a Forge issue",
		Action: func(opts core.Options) core.Result {
			org, repo, num := parseArgs(opts)
			if repo == "" || num == 0 {
				core.Print(nil, "usage: core-agent issue/get <repo> --number=N [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			issue, err := f.Issues.Get(ctx, forge.Params{"owner": org, "repo": repo, "index": fmtIndex(num)})
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "#%d %s", issue.Index, issue.Title)
			core.Print(nil, "  state:  %s", issue.State)
			core.Print(nil, "  url:    %s", issue.HTMLURL)
			if issue.Body != "" {
				core.Print(nil, "")
				core.Print(nil, "%s", issue.Body)
			}
			return core.Result{OK: true}
		},
	})

	c.Command("issue/list", core.Command{
		Description: "List Forge issues for a repo",
		Action: func(opts core.Options) core.Result {
			org, repo, _ := parseArgs(opts)
			if repo == "" {
				core.Print(nil, "usage: core-agent issue/list <repo> [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			issues, err := f.Issues.ListAll(ctx, forge.Params{"owner": org, "repo": repo})
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			for _, issue := range issues {
				core.Print(nil, "  #%-4d %-6s %s", issue.Index, issue.State, issue.Title)
			}
			if len(issues) == 0 {
				core.Print(nil, "  no issues")
			}
			return core.Result{OK: true}
		},
	})

	c.Command("issue/comment", core.Command{
		Description: "Comment on a Forge issue",
		Action: func(opts core.Options) core.Result {
			org, repo, num := parseArgs(opts)
			body := opts.String("body")
			if repo == "" || num == 0 || body == "" {
				core.Print(nil, "usage: core-agent issue/comment <repo> --number=N --body=\"text\" [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			comment, err := f.Issues.CreateComment(ctx, org, repo, num, body)
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "comment #%d created on %s/%s#%d", comment.ID, org, repo, num)
			return core.Result{OK: true}
		},
	})

	// --- Pull Requests ---

	c.Command("pr/get", core.Command{
		Description: "Get a Forge PR",
		Action: func(opts core.Options) core.Result {
			org, repo, num := parseArgs(opts)
			if repo == "" || num == 0 {
				core.Print(nil, "usage: core-agent pr/get <repo> --number=N [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			pr, err := f.Pulls.Get(ctx, forge.Params{"owner": org, "repo": repo, "index": fmtIndex(num)})
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "#%d %s", pr.Index, pr.Title)
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
		},
	})

	c.Command("pr/list", core.Command{
		Description: "List Forge PRs for a repo",
		Action: func(opts core.Options) core.Result {
			org, repo, _ := parseArgs(opts)
			if repo == "" {
				core.Print(nil, "usage: core-agent pr/list <repo> [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			prs, err := f.Pulls.ListAll(ctx, forge.Params{"owner": org, "repo": repo})
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			for _, pr := range prs {
				core.Print(nil, "  #%-4d %-6s %s → %s  %s", pr.Index, pr.State, pr.Head.Ref, pr.Base.Ref, pr.Title)
			}
			if len(prs) == 0 {
				core.Print(nil, "  no PRs")
			}
			return core.Result{OK: true}
		},
	})

	c.Command("pr/merge", core.Command{
		Description: "Merge a Forge PR",
		Action: func(opts core.Options) core.Result {
			org, repo, num := parseArgs(opts)
			method := opts.String("method")
			if method == "" {
				method = "merge"
			}
			if repo == "" || num == 0 {
				core.Print(nil, "usage: core-agent pr/merge <repo> --number=N [--method=merge|rebase|squash] [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			if err := f.Pulls.Merge(ctx, org, repo, num, method); err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "merged %s/%s#%d via %s", org, repo, num, method)
			return core.Result{OK: true}
		},
	})

	// --- Repositories ---

	c.Command("repo/get", core.Command{
		Description: "Get Forge repo info",
		Action: func(opts core.Options) core.Result {
			org, repo, _ := parseArgs(opts)
			if repo == "" {
				core.Print(nil, "usage: core-agent repo/get <repo> [--org=core]")
				return core.Result{OK: false}
			}

			f := newForgeClient()
			r, err := f.Repos.Get(ctx, forge.Params{"owner": org, "repo": repo})
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			core.Print(nil, "%s/%s", r.Owner.UserName, r.Name)
			core.Print(nil, "  description: %s", r.Description)
			core.Print(nil, "  default:     %s", r.DefaultBranch)
			core.Print(nil, "  private:     %v", r.Private)
			core.Print(nil, "  archived:    %v", r.Archived)
			core.Print(nil, "  url:         %s", r.HTMLURL)
			return core.Result{OK: true}
		},
	})

	c.Command("repo/list", core.Command{
		Description: "List Forge repos for an org",
		Action: func(opts core.Options) core.Result {
			org := opts.String("org")
			if org == "" {
				org = "core"
			}

			f := newForgeClient()
			repos, err := f.Repos.ListOrgRepos(ctx, org)
			if err != nil {
				core.Print(nil, "error: %v", err)
				return core.Result{Value: err, OK: false}
			}

			for _, r := range repos {
				archived := ""
				if r.Archived {
					archived = " (archived)"
				}
				core.Print(nil, "  %-30s %s%s", r.Name, r.Description, archived)
			}
			core.Print(nil, "\n  %d repos", len(repos))
			return core.Result{OK: true}
		},
	})
}
