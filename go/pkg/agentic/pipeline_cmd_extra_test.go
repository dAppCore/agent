// SPDX-License-Identifier: EUPL-1.2

// Happy + op-error coverage for the pipeline fix/* and epic/* CLI command
// wrappers. The existing pipeline tests cover only the missing-args guard;
// these drive the success path (mock the underlying pipelineFix*/pipelineEpic*
// var-op → assert OK + the mapped output) and the op-error path (op returns an
// error → wrapper returns a failed Result), without touching a real forge,
// git, or dispatch loop. Output is captured so the Print calls don't noise the
// test log.

package agentic

import (
	"context"
	"errors"
	"testing"

	core "dappco.re/go"
)

func pipelinePR(repo string, number int) core.Options {
	return core.NewOptions(
		core.Option{Key: "repo", Value: repo},
		core.Option{Key: "number", Value: number},
	)
}

// --- pipeline audit wrapper ---

// TestPipelineCmd_Audit_Good_MapsOutput — cmdPipelineAudit maps the repo option,
// calls the mocked pipelineAuditWithReader op, and surfaces the audits/created
// counts.
func TestPipelineCmd_Audit_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineAuditWithReader
	defer func() { pipelineAuditWithReader = orig }()
	pipelineAuditWithReader = func(_ *PrepSubsystem, _ context.Context, input PipelineAuditInput, _ *MetaReader) (PipelineAuditOutput, error) {
		core.AssertEqual(t, "go-io", input.Repo)
		return PipelineAuditOutput{
			Success: true, Org: "core", Repo: input.Repo,
			Audits:  []PipelineIssueRef{{Number: 3, Title: "Security audit"}},
			Created: []PipelineIssueRef{{Number: 21, Title: "Fix the SQLi"}},
		}, nil
	}

	captureStdout(t, func() {
		r := s.cmdPipelineAudit(core.NewOptions(core.Option{Key: "repo", Value: "go-io"}))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineAuditOutput)
		core.RequireTrue(t, ok)
		core.RequireTrue(t, len(out.Created) == 1)
		core.AssertEqual(t, 21, out.Created[0].Number)
	})
}

// TestPipelineCmd_Audit_Bad_OpErrors — an audit op error surfaces as a failed
// Result.
func TestPipelineCmd_Audit_Bad_OpErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineAuditWithReader
	defer func() { pipelineAuditWithReader = orig }()
	pipelineAuditWithReader = func(_ *PrepSubsystem, _ context.Context, _ PipelineAuditInput, _ *MetaReader) (PipelineAuditOutput, error) {
		return PipelineAuditOutput{}, errors.New("no forge token")
	}

	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineAudit(core.NewOptions(core.Option{Key: "repo", Value: "go-io"})).OK)
	})
}

// --- pipeline fix/* wrappers ---

// TestPipelineCmd_FixReviews_Good_MapsOutput — cmdPipelineFixReviews maps the
// repo+number options, calls the mocked pipelineFixReviews op, and surfaces its
// output.
func TestPipelineCmd_FixReviews_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixReviews
	defer func() { pipelineFixReviews = orig }()
	pipelineFixReviews = func(_ *PrepSubsystem, _ context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, 12, input.Number)
		return PipelineFixOutput{Success: true, Org: "core", Repo: input.Repo, Number: input.Number, Action: "commented"}, nil
	}

	captureStdout(t, func() {
		r := s.cmdPipelineFixReviews(pipelinePR("go-io", 12))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineFixOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "commented", out.Action)
	})
}

// TestPipelineCmd_FixReviews_Bad_OpErrors — an op error surfaces as a failed
// Result.
func TestPipelineCmd_FixReviews_Bad_OpErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixReviews
	defer func() { pipelineFixReviews = orig }()
	pipelineFixReviews = func(_ *PrepSubsystem, _ context.Context, _ PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{}, errors.New("forge down")
	}

	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineFixReviews(pipelinePR("go-io", 12)).OK)
	})
}

// TestPipelineCmd_FixConflicts_Good_MapsOutput — cmdPipelineFixConflicts maps
// + surfaces the mocked op output.
func TestPipelineCmd_FixConflicts_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixConflicts
	defer func() { pipelineFixConflicts = orig }()
	pipelineFixConflicts = func(_ *PrepSubsystem, _ context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{Success: true, Org: "core", Repo: input.Repo, Number: input.Number, Action: "rebased"}, nil
	}

	captureStdout(t, func() {
		r := s.cmdPipelineFixConflicts(pipelinePR("go-io", 7))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineFixOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "rebased", out.Action)
	})
}

// TestPipelineCmd_FixFormat_Good_MapsOutput — cmdPipelineFixFormat maps the
// commit/push/workspace options and surfaces the file/committed/pushed output.
func TestPipelineCmd_FixFormat_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixFormat
	defer func() { pipelineFixFormat = orig }()
	pipelineFixFormat = func(_ *PrepSubsystem, _ context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
		core.AssertTrue(t, input.Commit)
		return PipelineFixOutput{Success: true, Org: "core", Repo: input.Repo, Number: input.Number, Action: "formatted", Files: 3, Committed: true, Message: "gofmt"}, nil
	}

	captureStdout(t, func() {
		opts := core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "number", Value: 9},
			core.Option{Key: "commit", Value: true},
		)
		r := s.cmdPipelineFixFormat(opts)
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineFixOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, 3, out.Files)
		core.AssertTrue(t, out.Committed)
	})
}

// TestPipelineCmd_FixFormat_Bad_OpErrors — format op error → failed Result.
func TestPipelineCmd_FixFormat_Bad_OpErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixFormat
	defer func() { pipelineFixFormat = orig }()
	pipelineFixFormat = func(_ *PrepSubsystem, _ context.Context, _ PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{}, errors.New("gofmt failed")
	}

	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineFixFormat(pipelinePR("go-io", 9)).OK)
	})
}

// TestPipelineCmd_FixThreads_Good_MapsOutput — cmdPipelineFixThreads maps +
// surfaces the mocked op output.
func TestPipelineCmd_FixThreads_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineFixThreads
	defer func() { pipelineFixThreads = orig }()
	pipelineFixThreads = func(_ *PrepSubsystem, _ context.Context, input PipelineFixInput) (PipelineFixOutput, error) {
		return PipelineFixOutput{Success: true, Org: "core", Repo: input.Repo, Number: input.Number, Action: "resolved"}, nil
	}

	captureStdout(t, func() {
		r := s.cmdPipelineFixThreads(pipelinePR("go-io", 4))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineFixOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, "resolved", out.Action)
	})
}

// --- pipeline epic/* wrappers ---

// TestPipelineCmd_EpicCreate_Good_MapsOutput — cmdPipelineEpicCreate maps the
// repo/theme options, calls the mocked op, and surfaces the candidates/epics.
func TestPipelineCmd_EpicCreate_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineEpicCreate
	defer func() { pipelineEpicCreate = orig }()
	pipelineEpicCreate = func(_ *PrepSubsystem, _ context.Context, input PipelineEpicCreateInput) (PipelineEpicCreateOutput, error) {
		core.AssertEqual(t, "go-io", input.Repo)
		core.AssertEqual(t, "security", input.Theme)
		return PipelineEpicCreateOutput{
			Success: true, Org: "core", Repo: input.Repo,
			Epics: []PipelineEpicMeta{{Number: 11, Title: "Harden auth", Branch: "epic/security"}},
		}, nil
	}

	captureStdout(t, func() {
		opts := core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "theme", Value: "security"},
		)
		r := s.cmdPipelineEpicCreate(opts)
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineEpicCreateOutput)
		core.RequireTrue(t, ok)
		core.RequireTrue(t, len(out.Epics) == 1)
		core.AssertEqual(t, 11, out.Epics[0].Number)
	})
}

// TestPipelineCmd_EpicCreate_Bad_OpErrors — epic create op error → failed Result.
func TestPipelineCmd_EpicCreate_Bad_OpErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineEpicCreate
	defer func() { pipelineEpicCreate = orig }()
	pipelineEpicCreate = func(_ *PrepSubsystem, _ context.Context, _ PipelineEpicCreateInput) (PipelineEpicCreateOutput, error) {
		return PipelineEpicCreateOutput{}, errors.New("no candidates")
	}

	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineEpicCreate(core.NewOptions(core.Option{Key: "repo", Value: "go-io"})).OK)
	})
}

// TestPipelineCmd_EpicRun_Good_MapsOutput — cmdPipelineEpicRun maps the
// epic-number+agent options, calls the mocked op, surfaces the dispatched list.
func TestPipelineCmd_EpicRun_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineEpicRun
	defer func() { pipelineEpicRun = orig }()
	pipelineEpicRun = func(_ *PrepSubsystem, _ context.Context, input PipelineEpicRunInput) (PipelineEpicRunOutput, error) {
		core.AssertEqual(t, 11, input.EpicNumber)
		return PipelineEpicRunOutput{
			Success: true, Org: "core", Repo: input.Repo, EpicNumber: input.EpicNumber, Branch: "epic/x",
			Dispatched: []PipelineIssueRef{{Number: 21, Title: "Fix the queue"}},
		}, nil
	}

	captureStdout(t, func() {
		opts := core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "number", Value: 11},
			core.Option{Key: "agent", Value: "codex"},
		)
		r := s.cmdPipelineEpicRun(opts)
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineEpicRunOutput)
		core.RequireTrue(t, ok)
		core.RequireTrue(t, len(out.Dispatched) == 1)
		core.AssertEqual(t, 21, out.Dispatched[0].Number)
	})
}

// TestPipelineCmd_EpicSync_Good_MapsOutput — cmdPipelineEpicSync maps + surfaces
// the checked/total/updated counts.
func TestPipelineCmd_EpicSync_Good_MapsOutput(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineEpicSync
	defer func() { pipelineEpicSync = orig }()
	pipelineEpicSync = func(_ *PrepSubsystem, _ context.Context, _, repo string, number int, _ bool) (PipelineEpicSyncOutput, error) {
		return PipelineEpicSyncOutput{Success: true, Org: "core", Repo: repo, EpicNumber: number, Checked: 2, Total: 3, Updated: true}, nil
	}

	captureStdout(t, func() {
		r := s.cmdPipelineEpicSync(pipelinePR("go-io", 11))
		core.RequireTrue(t, r.OK)
		out, ok := r.Value.(PipelineEpicSyncOutput)
		core.RequireTrue(t, ok)
		core.AssertEqual(t, 2, out.Checked)
		core.AssertTrue(t, out.Updated)
	})
}

// TestPipelineCmd_EpicSync_Bad_OpErrors — sync op error → failed Result.
func TestPipelineCmd_EpicSync_Bad_OpErrors(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	orig := pipelineEpicSync
	defer func() { pipelineEpicSync = orig }()
	pipelineEpicSync = func(_ *PrepSubsystem, _ context.Context, _, _ string, _ int, _ bool) (PipelineEpicSyncOutput, error) {
		return PipelineEpicSyncOutput{}, errors.New("epic not found")
	}

	captureStdout(t, func() {
		core.AssertFalse(t, s.cmdPipelineEpicSync(pipelinePR("go-io", 11)).OK)
	})
}
