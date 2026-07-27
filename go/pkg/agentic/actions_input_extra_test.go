// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestActions_resumeInputFromOptions_Good(t *testing.T) {
	in := resumeInputFromOptions(core.NewOptions(
		core.Option{Key: "workspace", Value: "ws1"},
		core.Option{Key: "answer", Value: "yes"},
		core.Option{Key: "agent", Value: "codex"},
		core.Option{Key: "dry_run", Value: true},
	))
	core.AssertEqual(t, "ws1", in.Workspace)
	core.AssertEqual(t, "yes", in.Answer)
	core.AssertEqual(t, "codex", in.Agent)
	core.AssertTrue(t, in.DryRun)
}

func TestActions_scanInputFromOptions_Good(t *testing.T) {
	in := scanInputFromOptions(core.NewOptions(
		core.Option{Key: "org", Value: "lthn"},
		core.Option{Key: "limit", Value: 10},
	))
	core.AssertEqual(t, "lthn", in.Org)
	core.AssertEqual(t, 10, in.Limit)
}

func TestActions_watchInputFromOptions_Good_SingleWorkspaceFallback(t *testing.T) {
	in := watchInputFromOptions(core.NewOptions(
		core.Option{Key: "workspace", Value: "ws1"},
		core.Option{Key: "timeout", Value: 30},
	))
	core.AssertEqual(t, []string{"ws1"}, in.Workspaces)
	core.AssertEqual(t, 30, in.Timeout)
}

func TestActions_mirrorInputFromOptions_Good(t *testing.T) {
	in := mirrorInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "go-io"},
		core.Option{Key: "max_files", Value: 5},
	))
	core.AssertEqual(t, "go-io", in.Repo)
	core.AssertEqual(t, 5, in.MaxFiles)
}
