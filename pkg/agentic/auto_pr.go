// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
)

// autoCreatePR pushes the agent's branch and creates a PR on Forge
// if the agent made any commits beyond the initial clone.
func (s *PrepSubsystem) autoCreatePR(wsDir string) {
	result := ReadStatusResult(wsDir)
	st, ok := workspaceStatusValue(result)
	if !ok || st.Branch == "" || st.Repo == "" {
		return
	}

	ctx := context.Background()
	repoDir := WorkspaceRepoDir(wsDir)
	process := s.Core().Process()

	// PRs target dev — agents never merge directly to main
	base := "dev"

	r := process.RunIn(ctx, repoDir, "git", "log", "--oneline", core.Concat("origin/", base, "..HEAD"))
	if !r.OK {
		return
	}
	out := core.Trim(r.Value.(string))
	if out == "" {
		return
	}

	commitCount := len(core.Split(out, "\n"))

	org := st.Org
	if org == "" {
		org = "core"
	}

	// Push the branch to forge
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, st.Repo)
	if !process.RunIn(ctx, repoDir, "git", "push", forgeRemote, st.Branch).OK {
		if result := ReadStatusResult(wsDir); result.OK {
			st2, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			st2.Question = "PR push failed"
			writeStatusResult(wsDir, st2)
		}
		return
	}

	// Create PR via Forge API
	title := core.Sprintf("[agent/%s] %s", st.Agent, truncate(st.Task, 60))
	body := s.buildAutoPRBody(st, commitCount)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	prURL, _, err := s.forgeCreatePR(ctx, org, st.Repo, st.Branch, base, title, body)
	if err != nil {
		if result := ReadStatusResult(wsDir); result.OK {
			st2, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			st2.Question = core.Sprintf("PR creation failed: %v", err)
			writeStatusResult(wsDir, st2)
		}
		return
	}

	// Update status with PR URL
	if result := ReadStatusResult(wsDir); result.OK {
		st2, ok := workspaceStatusValue(result)
		if !ok {
			return
		}
		st2.PRURL = prURL
		writeStatusResult(wsDir, st2)
	}
}

func (s *PrepSubsystem) buildAutoPRBody(st *WorkspaceStatus, commits int) string {
	b := core.NewBuilder()
	b.WriteString("## Task\n\n")
	b.WriteString(st.Task)
	b.WriteString("\n\n")
	if st.Issue > 0 {
		b.WriteString(core.Sprintf("Closes #%d\n\n", st.Issue))
	}
	b.WriteString(core.Sprintf("**Agent:** %s\n", st.Agent))
	b.WriteString(core.Sprintf("**Commits:** %d\n", commits))
	b.WriteString(core.Sprintf("**Branch:** `%s`\n", st.Branch))
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
