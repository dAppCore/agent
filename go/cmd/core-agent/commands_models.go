// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lemma"
)

// CLI surface for managing model downloads on the local lthn-mlx
// serve via /v1/admin/models/*.
//
//	core-agent models-download --repo=lthn/lemer-lite                # kick + poll
//	core-agent models-download --repo=lthn/lemer-lite --no-wait      # kick + print job_id
//	core-agent models-job --id=dl-job-42                             # poll an existing job
//	core-agent models-list                                           # loaded models (no auth needed)
//
// Per the binary-is-the-model rule, every fetch lands in the engine's
// standard models dir — caller doesn't pick the destination. The
// upstream allowlist gates which repos can be fetched.

const (
	modelsPollInterval = 2 * time.Second
	modelsPollTimeout  = 60 * time.Minute
)

// modelsDownload kicks an async HF fetch. Default behaviour polls
// until the job lands in a terminal state and prints a final summary;
// --no-wait fires-and-forgets and prints the job id for separate
// monitoring via `models-job --id=`.
func (commands applicationCommandSet) modelsDownload(opts core.Options) core.Result {
	repo := opts.String("repo")
	if repo == "" {
		applicationPrint("models-download: --repo=<huggingface-id> required")
		return core.Result{}
	}
	revision := opts.String("revision")
	noWait := opts.Bool("no-wait")

	admin, ok := buildAdmin(opts)
	if !ok {
		return core.Result{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	jobID, err := admin.Download(ctx, lemma.DownloadRequest{
		RepoID:   repo,
		Revision: revision,
	})
	if err != nil {
		applicationPrint("models-download: %v", err)
		return core.Result{}
	}
	applicationPrint("models-download: queued job %s for %s", jobID, repo)
	if noWait {
		applicationPrint("  poll: core-agent models-job --id=%s", jobID)
		return core.Result{OK: true}
	}

	pollCtx, pollCancel := context.WithTimeout(context.Background(), modelsPollTimeout)
	defer pollCancel()
	return pollDownload(pollCtx, admin, jobID)
}

// modelsJob is the standalone poll command — read the status of an
// in-flight job kicked by an earlier --no-wait download or by an
// unrelated client (the lthn.ai pairing-dashboard sibling, etc).
func (commands applicationCommandSet) modelsJob(opts core.Options) core.Result {
	jobID := opts.String("id")
	if jobID == "" {
		applicationPrint("models-job: --id=<job-id> required")
		return core.Result{}
	}
	admin, ok := buildAdmin(opts)
	if !ok {
		return core.Result{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	js, err := admin.DownloadJob(ctx, jobID)
	if err != nil {
		applicationPrint("models-job: %v", err)
		return core.Result{}
	}
	printDownloadJob(js)
	return core.Result{OK: true}
}

// pollDownload loops on DownloadJob until the job hits a terminal
// state. Prints incremental progress per tick — operators want to see
// movement on a 30GB pull, not silent staring.
func pollDownload(ctx context.Context, admin *lemma.Admin, jobID string) core.Result {
	lastProgress := -1
	for {
		select {
		case <-ctx.Done():
			applicationPrint("models-download: timeout waiting for job %s", jobID)
			return core.Result{}
		default:
		}
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		js, err := admin.DownloadJob(callCtx, jobID)
		cancel()
		if err != nil {
			applicationPrint("models-download: poll: %v", err)
			return core.Result{}
		}
		if js.Progress != lastProgress {
			applicationPrint("  [%s] %d%%  bytes=%d", js.Status, js.Progress, js.Bytes)
			lastProgress = js.Progress
		}
		switch js.Status {
		case "done":
			applicationPrint("models-download: done — %s", js.Path)
			return core.Result{OK: true}
		case "failed":
			applicationPrint("models-download: failed — %s", js.Error)
			return core.Result{}
		}
		select {
		case <-ctx.Done():
			return core.Result{}
		case <-time.After(modelsPollInterval):
		}
	}
}

// printDownloadJob pretty-prints a single job snapshot. Shared by
// models-job + standalone status reads.
func printDownloadJob(js lemma.DownloadJobStatus) {
	applicationPrint("job %s", js.JobID)
	applicationPrint("  status:   %s", js.Status)
	if js.RepoID != "" {
		applicationPrint("  repo:     %s (revision=%s)", js.RepoID, js.Revision)
	}
	applicationPrint("  progress: %d%%  bytes=%d", js.Progress, js.Bytes)
	if js.Path != "" {
		applicationPrint("  path:     %s", js.Path)
	}
	if js.Error != "" {
		applicationPrint("  error:    %s", js.Error)
	}
}
