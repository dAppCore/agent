// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
)

// s.cleanupBranch(context.Background(), "core/go-io", "agent/fix-tests")
func (s *PrepSubsystem) cleanupBranch(ctx context.Context, repo, branch string) core.Result {
	if s == nil {
		return core.Result{Value: core.E("cleanupBranch", "prep subsystem is required", nil), OK: false}
	}
	if s.forgeToken == "" {
		return core.Result{Value: core.E("cleanupBranch", "no Forge token configured", nil), OK: false}
	}
	if s.forge == nil {
		return core.Result{Value: core.E("cleanupBranch", "forge client is not configured", nil), OK: false}
	}

	org, repoName, ok := cleanupBranchRepo(repo)
	if !ok || repoName == "" {
		return core.Result{Value: core.E("cleanupBranch", "repo must be <repo> or <org/repo>", nil), OK: false}
	}

	branch = cleanupBranchName(branch)
	if branch == "" {
		return core.Result{Value: core.E("cleanupBranch", "branch is required", nil), OK: false}
	}
	if protectedBranch(branch) {
		return core.Result{Value: core.E("cleanupBranch", core.Concat("refusing to delete protected branch ", branch), nil), OK: false}
	}

	deleteResult := s.forge.deleteBranch(ctx, org, repoName, branch)
	deleteErr := forgeResultError(deleteResult)
	if deleteErr != nil && !isForgeNotFound(deleteErr) {
		return core.Result{Value: core.E("cleanupBranch", core.Concat("failed to delete branch ", branch), deleteErr), OK: false}
	}
	return core.Result{OK: true}
}

// s.cleanupWorkspaceBranch(context.Background(), "/srv/.core/workspace/core/go-io/task-42")
func (s *PrepSubsystem) cleanupWorkspaceBranch(ctx context.Context, workspace string) core.Result {
	workspaceDir := core.Trim(workspace)
	if workspaceDir == "" {
		return core.Result{OK: true}
	}
	if !fs.IsDir(workspaceDir) {
		candidate := core.JoinPath(WorkspaceRoot(), workspaceDir)
		if !fs.IsDir(candidate) {
			return core.Result{OK: true}
		}
		workspaceDir = candidate
	}

	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		return core.Result{OK: true}
	}
	if workspaceStatus.Repo == "" || workspaceStatus.Branch == "" {
		return core.Result{OK: true}
	}
	if workspaceStatus.PRURL == "" && workspaceStatus.Status != "merged" {
		return core.Result{OK: true}
	}

	return s.cleanupBranch(ctx, cleanupRepoRef(workspaceStatus.Org, workspaceStatus.Repo), workspaceStatus.Branch)
}

// repo := cleanupRepoRef("core", "go-io")
func cleanupRepoRef(org, repo string) string {
	repo = core.Trim(repo)
	if repo == "" {
		return ""
	}
	if core.Contains(repo, "/") {
		return repo
	}
	org = core.Trim(org)
	if org == "" {
		org = "core"
	}
	return core.Concat(org, "/", repo)
}

// org, repo, ok := cleanupBranchRepo("core/go-io")
func cleanupBranchRepo(repo string) (string, string, bool) {
	repo = core.Trim(repo)
	if repo == "" {
		return "", "", false
	}

	parts := core.Split(repo, "/")
	if len(parts) == 1 {
		return "core", parts[0], parts[0] != ""
	}
	if len(parts) == 2 {
		return parts[0], parts[1], parts[0] != "" && parts[1] != ""
	}
	return "", "", false
}

// branch := cleanupBranchName("refs/heads/agent/fix-tests")
func cleanupBranchName(branch string) string {
	branch = core.Trim(branch)
	branch = core.TrimPrefix(branch, "refs/heads/")
	return branch
}

// ok := protectedBranch("main")
func protectedBranch(branch string) bool {
	switch cleanupBranchName(branch) {
	case "main", "dev", "master":
		return true
	default:
		return false
	}
}
