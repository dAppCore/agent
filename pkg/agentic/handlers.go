// SPDX-License-Identifier: EUPL-1.2

// IPC handlers for the agent completion pipeline.
// Registered via RegisterHandlers() — breaks the monolith dispatch goroutine
// into discrete, testable steps connected by Core IPC messages.

package agentic

import (
	"dappco.re/go/agent/pkg/messages"
	core "dappco.re/go/core"
)

// RegisterHandlers registers the post-completion pipeline as discrete IPC handlers.
// Each handler listens for a specific message and emits the next in the chain:
//
//	AgentCompleted → QA handler → QAResult
//	QAResult{Passed} → PR handler → PRCreated
//	PRCreated → Verify handler → PRMerged | PRNeedsReview
//	AgentCompleted → Ingest handler (findings → issues)
//	AgentCompleted → Poke handler (drain queue)
//
//	agentic.RegisterHandlers(c, prep)
func RegisterHandlers(c *core.Core, s *PrepSubsystem) {
	// QA: run build+test on completed workspaces
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		ev, ok := msg.(messages.AgentCompleted)
		if !ok || ev.Status != "completed" {
			return core.Result{OK: true}
		}
		wsDir := resolveWorkspace(ev.Workspace)
		if wsDir == "" {
			return core.Result{OK: true}
		}

		passed := s.runQA(wsDir)
		if !passed {
			// Update status to failed
			if st, err := ReadStatus(wsDir); err == nil {
				st.Status = "failed"
				st.Question = "QA check failed — build or tests did not pass"
				writeStatus(wsDir, st)
			}
		}

		c.ACTION(messages.QAResult{
			Workspace: ev.Workspace,
			Repo:      ev.Repo,
			Passed:    passed,
		})
		return core.Result{OK: true}
	})

	// Auto-PR: create PR on QA pass
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		ev, ok := msg.(messages.QAResult)
		if !ok || !ev.Passed {
			return core.Result{OK: true}
		}
		wsDir := resolveWorkspace(ev.Workspace)
		if wsDir == "" {
			return core.Result{OK: true}
		}

		s.autoCreatePR(wsDir)
		return core.Result{OK: true}
	})

	// Auto-verify: verify and merge after PR creation
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		ev, ok := msg.(messages.QAResult)
		if !ok || !ev.Passed {
			return core.Result{OK: true}
		}
		wsDir := resolveWorkspace(ev.Workspace)
		if wsDir == "" {
			return core.Result{OK: true}
		}

		s.autoVerifyAndMerge(wsDir)
		return core.Result{OK: true}
	})

	// Ingest: create issues from agent findings
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		ev, ok := msg.(messages.AgentCompleted)
		if !ok {
			return core.Result{OK: true}
		}
		wsDir := resolveWorkspace(ev.Workspace)
		if wsDir == "" {
			return core.Result{OK: true}
		}

		s.ingestFindings(wsDir)
		return core.Result{OK: true}
	})

	// Poke: drain queue after any completion
	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
		if _, ok := msg.(messages.AgentCompleted); ok {
			s.Poke()
		}
		if _, ok := msg.(messages.PokeQueue); ok {
			s.drainQueue()
		}
		return core.Result{OK: true}
	})
}

// resolveWorkspace converts a workspace name back to the full path.
//
//	resolveWorkspace("core/go-io/task-5") → "/Users/snider/Code/.core/workspace/core/go-io/task-5"
func resolveWorkspace(name string) string {
	wsRoot := WorkspaceRoot()
	path := core.JoinPath(wsRoot, name)
	if fs.IsDir(path) {
		return path
	}
	return ""
}
