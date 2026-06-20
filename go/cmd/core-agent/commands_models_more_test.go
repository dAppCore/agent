// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// modelsStubServer answers the /v1/admin/models/download route (POST kicks a
// job, GET polls one) with the supplied JSON. An empty body string makes the
// route 500 so the handler's error branch is exercised.
func modelsStubServer(t *testing.T, postBody, getBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/admin/models/download" {
			http.Error(w, "no stub for "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		body := getBody
		if r.Method == http.MethodPost {
			body = postBody
		}
		if body == "" {
			http.Error(w, "stub error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

// TestModels_modelsDownload_Good_NoWait — --repo + --no-wait queues a job and
// prints the job id + poll hint without entering the poll loop.
func TestModels_modelsDownload_Good_NoWait(t *testing.T) {
	srv := modelsStubServer(t, `{"job_id": "dl-42"}`, "")
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.modelsDownload(stubAdminOpts(srv.URL,
			core.Option{Key: "repo", Value: "lthn/lemer-lite"},
			core.Option{Key: "revision", Value: "main"},
			core.Option{Key: "no-wait", Value: true},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "queued job dl-42")
	core.AssertContains(t, out, "models-job --id=dl-42")
}

// TestModels_modelsDownload_Bad_DaemonError — --repo set but the download
// route 500s prints the error and returns non-OK (without polling).
func TestModels_modelsDownload_Bad_DaemonError(t *testing.T) {
	srv := modelsStubServer(t, "", "") // POST 500s
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.modelsDownload(stubAdminOpts(srv.URL,
			core.Option{Key: "repo", Value: "lthn/lemer-lite"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "models-download:")
}

// TestModels_modelsJob_Good_Prints — --id + a stub job status renders the
// job snapshot and returns OK.
func TestModels_modelsJob_Good_Prints(t *testing.T) {
	srv := modelsStubServer(t, "", `{
		"job_id": "dl-42",
		"status": "done",
		"repo_id": "lthn/lemer-lite",
		"revision": "main",
		"progress": 100,
		"bytes": 2048,
		"path": "/Lethean/models/lemer-lite"
	}`)
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.modelsJob(stubAdminOpts(srv.URL, core.Option{Key: "id", Value: "dl-42"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "dl-42")
	core.AssertContains(t, out, "done")
	core.AssertContains(t, out, "100%")
	core.AssertContains(t, out, "/Lethean/models/lemer-lite")
}

// TestModels_modelsJob_Bad_DaemonError — --id set but the poll route 500s
// prints the error and returns non-OK.
func TestModels_modelsJob_Bad_DaemonError(t *testing.T) {
	srv := modelsStubServer(t, "", "") // GET 500s
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.modelsJob(stubAdminOpts(srv.URL, core.Option{Key: "id", Value: "dl-42"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "models-job:")
}
