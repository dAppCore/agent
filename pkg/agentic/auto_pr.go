// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
)

// autoCreatePR pushes the agent's branch and creates a PR on Forge
// if the agent made any commits beyond the initial clone.
func (s *PrepSubsystem) autoCreatePR(workspaceDir string) {
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok || workspaceStatus.Branch == "" || workspaceStatus.Repo == "" {
		return
	}

	ctx := context.Background()
	repoDir := WorkspaceRepoDir(workspaceDir)
	process := s.Core().Process()

	// PRs target dev — agents never merge directly to main
	base := "dev"

	processResult := process.RunIn(ctx, repoDir, "git", "log", "--oneline", core.Concat("origin/", base, "..HEAD"))
	if !processResult.OK {
		return
	}
	out := core.Trim(processResult.Value.(string))
	if out == "" {
		return
	}

	commitCount := len(core.Split(out, "\n"))

	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}

	// Push the branch to forge
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, workspaceStatus.Repo)
	if !process.RunIn(ctx, repoDir, "git", "push", forgeRemote, workspaceStatus.Branch).OK {
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatusUpdate, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			workspaceStatusUpdate.Question = "PR push failed"
			writeStatusResult(workspaceDir, workspaceStatusUpdate)
		}
		return
	}

	// Create PR via Forge API
	title := core.Sprintf("[agent/%s] %s", workspaceStatus.Agent, truncate(workspaceStatus.Task, 60))
	body := s.buildAutoPRBody(workspaceStatus, commitCount)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prURL, _, err := s.forgeCreatePR(ctx, org, workspaceStatus.Repo, workspaceStatus.Branch, base, title, body)
	if err != nil {
		if result := ReadStatusResult(workspaceDir); result.OK {
			workspaceStatusUpdate, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			workspaceStatusUpdate.Question = core.Sprintf("PR creation failed: %v", err)
			writeStatusResult(workspaceDir, workspaceStatusUpdate)
		}
		return
	}

	// Update status with PR URL
	if result := ReadStatusResult(workspaceDir); result.OK {
		workspaceStatusUpdate, ok := workspaceStatusValue(result)
		if !ok {
			return
		}
		workspaceStatusUpdate.PRURL = prURL
		writeStatusResult(workspaceDir, workspaceStatusUpdate)
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

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return core.Concat(s[:max], "...")
}
