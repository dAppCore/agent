// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestDispatchSyncInputFromOptions_Good_AllFields(t *testing.T) {
	in := dispatchSyncInputFromOptions(core.NewOptions(
		core.Option{Key: "org", Value: "core"},
		core.Option{Key: "repo", Value: "agent"},
		core.Option{Key: "agent", Value: "opencode:opencode-go/deepseek-v4-pro"},
		core.Option{Key: "task", Value: "add tests"},
		core.Option{Key: "branch", Value: "test-coverage"},
		core.Option{Key: "issue", Value: 42},
	))

	core.AssertEqual(t, "core", in.Org)
	core.AssertEqual(t, "agent", in.Repo)
	core.AssertEqual(t, "opencode:opencode-go/deepseek-v4-pro", in.Agent)
	core.AssertEqual(t, "add tests", in.Task)
	core.AssertEqual(t, "test-coverage", in.Branch)
	core.AssertEqual(t, 42, in.Issue)
}

func TestDispatchSyncInputFromOptions_Bad_OptionalFieldsZeroWhenAbsent(t *testing.T) {
	in := dispatchSyncInputFromOptions(core.NewOptions(
		core.Option{Key: "repo", Value: "agent"},
		core.Option{Key: "task", Value: "x"},
	))

	// No --branch / --issue → zero values (prep then requires one of them).
	core.AssertEqual(t, "", in.Branch)
	core.AssertEqual(t, 0, in.Issue)
}

func TestDispatchSyncInputFromOptions_Ugly_RepoFromPositionalArg(t *testing.T) {
	// repo falls back to the "_arg" positional when --repo is absent; branch
	// still maps from its flag.
	in := dispatchSyncInputFromOptions(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "branch", Value: "b"},
	))

	core.AssertEqual(t, "go-io", in.Repo)
	core.AssertEqual(t, "b", in.Branch)
}
