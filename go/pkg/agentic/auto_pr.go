// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/messages"
)

// s.autoCreatePR("/srv/.core/workspace/core/go-io/task-5")
func (s *PrepSubsystem) autoCreatePR(workspaceDir string) {
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok || workspaceStatus.Branch == "" || workspaceStatus.Repo == "" {
		return
	}

	ctx := context.Background()
	repoDir := WorkspaceRepoDir(workspaceDir)
	process := s.Core().Process()

	defaultBranch := "dev"

	processResult := process.RunIn(ctx, repoDir, "git", `log`, "--oneline", core.Concat("origin/", defaultBranch, "..HEAD"))
	if !processResult.OK {
		return
	}
	commitLogOutput := core.Trim(processResult.Value.(string))
	if commitLogOutput == "" {
		return
	}

	commitCount := len(core.Split(commitLogOutput, "\n"))

	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}

	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, workspaceStatus.Repo)
	if !process.RunIn(ctx, repoDir, "git", "push", forgeRemote, workspaceStatus.Branch).OK {
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatusUpdate, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			workspaceStatusUpdate.Question = "PR push failed"
			if result := writeStatusResult(workspaceDir, workspaceStatusUpdate); !result.OK {
				core.Warn("agentic.autoPR: failed to record push failure", "reason", result.Error())
			}
		}
		return
	}
	if s.ServiceRuntime != nil {
		if result := s.Core().ACTION(messages.WorkspacePushed{
			Repo:   workspaceStatus.Repo,
			Branch: workspaceStatus.Branch,
			Org:    org,
		}); !result.OK {
			core.Warn("agentic.autoPR: workspace push notification failed", "reason", result.Error())
		}
	}

	title := core.Sprintf("[agent/%s] %s", workspaceStatus.Agent, truncate(workspaceStatus.Task, 60))
	body := s.buildAutoPRBody(workspaceStatus, commitCount)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pullRequestURL, _, err := forgeCreatePR(s, ctx, org, workspaceStatus.Repo, workspaceStatus.Branch, defaultBranch, title, body)
	if err != nil {
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatusUpdate, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			workspaceStatusUpdate.Question = core.Sprintf("PR creation failed: %v", err)
			if result := writeStatusResult(workspaceDir, workspaceStatusUpdate); !result.OK {
				core.Warn("agentic.autoPR: failed to record PR failure", "reason", result.Error())
			}
		}
		return
	}

	s.cleanupForgeBranch(ctx, repoDir, forgeRemote, workspaceStatus.Branch)

	if result := ReadStatusResult(workspaceDir); result.OK {
		workspaceStatusUpdate, ok := workspaceStatusValue(result)
		if !ok {
			return
		}
		workspaceStatusUpdate.PRURL = pullRequestURL
		if r := writeStatusResult(workspaceDir, workspaceStatusUpdate); !r.OK {
			core.Warn("agentic: failed to record PR URL on status", "workspace", workspaceDir, "reason", r.Value)
		}
	}
}

func (s *PrepSubsystem) buildAutoPRBody(workspaceStatus *WorkspaceStatus, commits int) string {
	b := core.NewBuilder()
	b.WriteString("## Task\n\n")
	b.WriteString(workspaceStatus.Task)
	b.WriteString("\n\n")
	if workspaceStatus.Issue > 0 {
		b.WriteString(core.Sprintf("Closes #%d\n\n", workspaceStatus.Issue))
	}
	b.WriteString(core.Sprintf("**Agent:** %s\n", workspaceStatus.Agent))
	b.WriteString(core.Sprintf("**Commits:** %d\n", commits))
	b.WriteString(core.Sprintf("**Branch:** `%s`\n", workspaceStatus.Branch))
	b.WriteString("\n---\n")
	b.WriteString("Auto-created by core-agent dispatch system.\n")
	b.WriteString("Co-Authored-By: Virgil <virgil@lethean.io>\n")
	return b.String()
}

// cleanupForgeBranch removes an agent branch from the Forge remote after the PR is published.
//
//	s.cleanupForgeBranch(context.Background(), "/workspace/repo", "ssh://git@forge.lthn.ai:2223/core/go-io.git", "agent/fix-tests")
func (s *PrepSubsystem) cleanupForgeBranch(ctx context.Context, repoDir, remote, branch string) bool {
	if repoDir == "" || remote == "" || branch == "" {
		return false
	}
	if s == nil || s.ServiceRuntime == nil {
		return false
	}

	result := s.Core().Process().RunIn(ctx, repoDir, "git", "push", remote, "--delete", branch)
	return result.OK
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return core.Concat(s[:max], "...")
}
