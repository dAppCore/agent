// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lemma"
)

// TestModels_printDownloadJob_Good — the download-job printer renders all
// populated fields.
func TestModels_printDownloadJob_Good(t *testing.T) {
	out := captureStdout(t, func() {
		printDownloadJob(lemma.DownloadJobStatus{
			JobID: "j1", Status: "running", RepoID: "repo", Revision: "main",
			Progress: 50, Bytes: 1024, Path: "/x", Error: "boom",
		})
	})
	core.AssertContains(t, out, "j1")
	core.AssertContains(t, out, "running")
	core.AssertContains(t, out, "repo")
	core.AssertContains(t, out, "50%")
	core.AssertContains(t, out, "/x")
	core.AssertContains(t, out, "boom")
}

// TestModels_Handlers_NoDaemon — models download + opencode-models fail without
// a reachable daemon rather than panicking.
func TestModels_Handlers_NoDaemon(t *testing.T) {
	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	captureStdout(t, func() {
		core.AssertFalse(t, cmds.modelsDownload(core.NewOptions()).OK)
		core.AssertFalse(t, cmds.modelsJob(core.NewOptions()).OK)
		core.AssertFalse(t, cmds.opencodeModels(core.NewOptions()).OK)
	})
}
