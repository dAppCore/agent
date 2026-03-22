// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os/exec"
	"time"

	core "dappco.re/go/core"
)

// autoCreatePR pushes the agent's branch and creates a PR on Forge
// if the agent made any commits beyond the initial clone.
func (s *PrepSubsystem) autoCreatePR(wsDir string) {
	st, err := readStatus(wsDir)
	if err != nil || st.Branch == "" || st.Repo == "" {
		return
	}

	srcDir := core.JoinPath(wsDir, "src")

	// Detect default branch for this repo
	base := DefaultBranch(srcDir)

	// Check if there are commits on the branch beyond the default branch
	diffCmd := exec.Command("git", "log", "--oneline", "origin/"+base+"..HEAD")
	diffCmd.Dir = srcDir
	out, err := diffCmd.Output()
	if err != nil || len(core.Trim(string(out))) == 0 {
		// No commits — nothing to PR
		return
	}

	commitCount := len(core.Split(core.Trim(string(out)), "\n"))

	// Get the repo's forge remote URL to extract org/repo
	org := st.Org
	if org == "" {
		org = "core"
	}

	// Push the branch to forge
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, st.Repo)
	pushCmd := exec.Command("git", "push", forgeRemote, st.Branch)
	pushCmd.Dir = srcDir
	if pushErr := pushCmd.Run(); pushErr != nil {
		// Push failed — update status with error but don't block
		if st2, err := readStatus(wsDir); err == nil {
			st2.Question = core.Sprintf("PR push failed: %v", pushErr)
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
		if st2, err := readStatus(wsDir); err == nil {
			st2.Question = core.Sprintf("PR creation failed: %v", err)
			writeStatus(wsDir, st2)
		}
		return
	}

	// Update status with PR URL
	if st2, err := readStatus(wsDir); err == nil {
		st2.PRURL = prURL
		writeStatus(wsDir, st2)
	}
}

func (s *PrepSubsystem) buildAutoPRBody(st *WorkspaceStatus, commits int) string {
	b := core.NewBuilder()
	b.WriteString("## Task\n\n")
	b.WriteString(st.Task)
	b.WriteString("\n\n")
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
	return s[:max] + "..."
}
