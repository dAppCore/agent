// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

// TestPipelineEpicCov_CmdStatus_Good_PrintsEpicAndChildren — the epic-status
// command wrapper reads the epic meta, prints the epic header plus a checkbox
// line per child, and returns the typed status output. HTTP-only path.
func TestPipelineEpicCov_CmdStatus_Good_PrintsEpicAndChildren(t *testing.T) {
	repo := newPipelineTestRepo()
	repo.Issues[20] = &pipelineTestIssue{
		Number: 20,
		Title:  "epic(go-io): security pipeline",
		State:  "open",
		Labels: []string{"agentic", "epic", "security"},
		Body:   "## Overview\n\nEpic branch: `epic/20-security`\n\n## Child Issues\n\n- [x] #10 Validate tokens\n- [ ] #11 Sanitize input\n",
	}
	repo.Issues[10] = &pipelineTestIssue{Number: 10, Title: "Validate tokens", State: "closed"}
	repo.Issues[11] = &pipelineTestIssue{Number: 11, Title: "Sanitize input", State: "open"}
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": repo})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineEpicStatus(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "20"},
		))
	})

	core.RequireTrue(t, result.OK)
	typed, ok := result.Value.(PipelineEpicStatusOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, 20, typed.Epic.Number)
	core.AssertEqual(t, "epic/20-security", typed.Epic.Branch)
	core.AssertLen(t, typed.Epic.Children, 2)
	core.AssertContains(t, output, "epic:    #20 epic(go-io): security pipeline")
	core.AssertContains(t, output, "branch:  epic/20-security")
	core.AssertContains(t, output, "child:   2")
	core.AssertContains(t, output, "[x] #10")
	core.AssertContains(t, output, "[ ] #11")
}

// TestPipelineEpicCov_CmdStatus_Bad_MissingRepoAndNumber — the wrapper prints
// usage and returns an error envelope when neither repo nor number is supplied.
func TestPipelineEpicCov_CmdStatus_Bad_MissingRepoAndNumber(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineEpicStatus(core.NewOptions())
	})

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "repo and epic number are required")
	core.AssertContains(t, output, "usage: core-agent pipeline/epic/status")
}

// TestPipelineEpicCov_CmdStatus_Ugly_ReaderErrorPropagates — when the epic
// issue cannot be read the wrapper prints the error and fails.
func TestPipelineEpicCov_CmdStatus_Ugly_ReaderErrorPropagates(t *testing.T) {
	srv := newPipelineTestServer(t, map[string]*pipelineTestRepo{"go-io": newPipelineTestRepo()})
	s, _ := testPrepWithCore(t, srv)

	var result core.Result
	output := captureStdout(t, func() {
		result = s.cmdPipelineEpicStatus(core.NewOptions(
			core.Option{Key: "repo", Value: "go-io"},
			core.Option{Key: "_arg", Value: "999"},
		))
	})

	core.AssertFalse(t, result.OK)
	core.AssertContains(t, output, "error:")
}
