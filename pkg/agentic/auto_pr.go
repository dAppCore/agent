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
	st, err := ReadStatus(wsDir)
	if err != nil || st.Branch == "" || st.Repo == "" {
		return
	}

	ctx := context.Background()
	repoDir := WorkspaceRepoDir(wsDir)

	// PRs target dev — agents never merge directly to main
	base := "dev"

	out := s.gitOutput(ctx, repoDir, "log", "--oneline", core.Concat("origin/", base, "..HEAD"))
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
	if !s.gitCmdOK(ctx, repoDir, "push", forgeRemote, st.Branch) {
		if st2, err := ReadStatus(wsDir); err == nil {
			st2.Question = "PR push failed"
			writeStatus(wsDir, st2)
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
		if st2, err := ReadStatus(wsDir); err == nil {
			st2.Question = core.Sprintf("PR creation failed: %v", err)
			writeStatus(wsDir, st2)
		}
		return
	}

	// Update status with PR URL
	if st2, err := ReadStatus(wsDir); err == nil {
		st2.PRURL = prURL
		writeStatus(wsDir, st2)
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
