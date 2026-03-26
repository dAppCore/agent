// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_mirror tool ---

// MirrorInput is the input for agentic_mirror.
//
//	input := agentic.MirrorInput{Repo: "go-io", DryRun: true, MaxFiles: 50}
type MirrorInput struct {
	Repo     string `json:"repo,omitempty"`      // Specific repo, or empty for all
	DryRun   bool   `json:"dry_run,omitempty"`   // Preview without pushing
	MaxFiles int    `json:"max_files,omitempty"` // Max files per PR (default 50, CodeRabbit limit)
}

// MirrorOutput is the output for agentic_mirror.
//
//	out := agentic.MirrorOutput{Success: true, Count: 1, Synced: []agentic.MirrorSync{{Repo: "go-io"}}}
type MirrorOutput struct {
	Success bool         `json:"success"`
	Synced  []MirrorSync `json:"synced"`
	Skipped []string     `json:"skipped,omitempty"`
	Count   int          `json:"count"`
}

// MirrorSync records one repo sync.
//
//	sync := agentic.MirrorSync{Repo: "go-io", CommitsAhead: 3, FilesChanged: 12}
type MirrorSync struct {
	Repo         string `json:"repo"`
	CommitsAhead int    `json:"commits_ahead"`
	FilesChanged int    `json:"files_changed"`
	PRURL        string `json:"pr_url,omitempty"`
	Pushed       bool   `json:"pushed"`
	Skipped      string `json:"skipped,omitempty"`
}

func (s *PrepSubsystem) registerMirrorTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_mirror",
		Description: "Sync Forge repos to GitHub mirrors. Pushes Forge main to GitHub dev branch and creates a PR. Respects file count limits for CodeRabbit review.",
	}, s.mirror)
}

func (s *PrepSubsystem) mirror(ctx context.Context, _ *mcp.CallToolRequest, input MirrorInput) (*mcp.CallToolResult, MirrorOutput, error) {
	maxFiles := input.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}

	basePath := s.codePath
	if basePath == "" {
		basePath = core.JoinPath(core.Env("DIR_HOME"), "Code", "core")
	} else {
		basePath = core.JoinPath(basePath, "core")
	}

	// Build list of repos to sync
	var repos []string
	if input.Repo != "" {
		repos = []string{input.Repo}
	} else {
		repos = s.listLocalRepos(basePath)
	}

	var synced []MirrorSync
	var skipped []string

	for _, repo := range repos {
		repoDir := core.JoinPath(basePath, repo)

		// Check if github remote exists
		if !s.hasRemote(repoDir, "github") {
			skipped = append(skipped, core.Concat(repo, ": no github remote"))
			continue
		}

		// Fetch github to get current state
		s.gitCmdOK(ctx, repoDir, "fetch", "github")

		// Check how far ahead local default branch is vs github
		localBase := s.DefaultBranch(repoDir)
		ahead := s.commitsAhead(repoDir, "github/main", localBase)
		if ahead == 0 {
			continue // Already in sync
		}

		// Count files changed
		files := s.filesChanged(repoDir, "github/main", localBase)

		sync := MirrorSync{
			Repo:         repo,
			CommitsAhead: ahead,
			FilesChanged: files,
		}

		// Skip if too many files for one PR
		if files > maxFiles {
			sync.Skipped = core.Sprintf("%d files exceeds limit of %d", files, maxFiles)
			synced = append(synced, sync)
			continue
		}

		if input.DryRun {
			sync.Skipped = "dry run"
			synced = append(synced, sync)
			continue
		}

		// Ensure dev branch exists on GitHub
		s.ensureDevBranch(repoDir)

		// Push local main to github dev (explicit main, not HEAD)
		base := s.DefaultBranch(repoDir)
		if r := s.gitCmd(ctx, repoDir, "push", "github", core.Concat(base, ":refs/heads/dev"), "--force"); !r.OK {
			sync.Skipped = core.Sprintf("push failed: %s", r.Value)
			synced = append(synced, sync)
			continue
		}
		sync.Pushed = true

		// Create PR: dev → main on GitHub
		prURL, err := s.createGitHubPR(ctx, repoDir, repo, ahead, files)
		if err != nil {
			sync.Skipped = core.Sprintf("PR creation failed: %v", err)
		} else {
			sync.PRURL = prURL
		}

		synced = append(synced, sync)
	}

	return nil, MirrorOutput{
		Success: true,
		Synced:  synced,
		Skipped: skipped,
		Count:   len(synced),
	}, nil
}

// createGitHubPR creates a PR from dev → main using the gh CLI.
func (s *PrepSubsystem) createGitHubPR(ctx context.Context, repoDir, repo string, commits, files int) (string, error) {
	ghRepo := core.Sprintf("%s/%s", GitHubOrg(), repo)

	// Check if there's already an open PR from dev
	r := s.runCmd(ctx, repoDir, "gh", "pr", "list", "--repo", ghRepo, "--head", "dev", "--state", "open", "--json", "url", "--limit", "1")
	if r.OK {
		out := r.Value.(string)
		if core.Contains(out, "url") {
			if url := extractJSONField(out, "url"); url != "" {
				return url, nil
			}
		}
	}

	body := core.Sprintf(
		"## Forge → GitHub Sync\n\n**Commits:** %d\n**Files changed:** %d\n\nAutomated sync from Forge (forge.lthn.ai) to GitHub mirror.\nReview with CodeRabbit before merging.\n\n---\nCo-Authored-By: Virgil <virgil@lethean.io>",
		commits, files)

	title := core.Sprintf("[sync] %s: %d commits, %d files", repo, commits, files)

	r = s.runCmd(ctx, repoDir, "gh", "pr", "create",
		"--repo", ghRepo, "--head", "dev", "--base", "main",
		"--title", title, "--body", body)
	if !r.OK {
		return "", core.E("createGitHubPR", r.Value.(string), nil)
	}

	prOut := r.Value.(string)
	lines := core.Split(core.Trim(prOut), "\n")
	if len(lines) > 0 {
		return lines[len(lines)-1], nil
	}
	return "", nil
}

// ensureDevBranch creates the dev branch on GitHub if it doesn't exist.
func (s *PrepSubsystem) ensureDevBranch(repoDir string) {
	s.gitCmdOK(context.Background(), repoDir, "push", "github", "HEAD:refs/heads/dev")
}

// hasRemote checks if a git remote exists.
func (s *PrepSubsystem) hasRemote(repoDir, name string) bool {
	return s.gitCmdOK(context.Background(), repoDir, "remote", "get-url", name)
}

// commitsAhead returns how many commits HEAD is ahead of the ref.
func (s *PrepSubsystem) commitsAhead(repoDir, base, head string) int {
	out := s.gitOutput(context.Background(), repoDir, "rev-list", core.Concat(base, "..", head), "--count")
	return parseInt(out)
}

// filesChanged returns the number of files changed between two refs.
func (s *PrepSubsystem) filesChanged(repoDir, base, head string) int {
	out := s.gitOutput(context.Background(), repoDir, "diff", "--name-only", core.Concat(base, "..", head))
	if out == "" {
		return 0
	}
	return len(core.Split(out, "\n"))
}

// listLocalRepos returns repo names that exist as directories in basePath.
func (s *PrepSubsystem) listLocalRepos(basePath string) []string {
	paths := core.PathGlob(core.JoinPath(basePath, "*"))
	var repos []string
	for _, p := range paths {
		name := core.PathBase(p)
		if !fs.IsDir(p) {
			continue
		}
		// Must have a .git directory
		if fs.IsDir(core.JoinPath(basePath, name, ".git")) {
			repos = append(repos, name)
		}
	}
	return repos
}

// extractJSONField extracts a simple string field from JSON array output.
func extractJSONField(jsonStr, field string) string {
	if jsonStr == "" || field == "" {
		return ""
	}

	var list []map[string]any
	if r := core.JSONUnmarshalString(jsonStr, &list); r.OK {
		for _, item := range list {
			if value, ok := item[field].(string); ok {
				return value
			}
		}
	}

	var item map[string]any
	if r := core.JSONUnmarshalString(jsonStr, &item); !r.OK {
		return ""
	}

	value, _ := item[field].(string)
	return value
}
