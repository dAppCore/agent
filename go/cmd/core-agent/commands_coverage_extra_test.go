// SPDX-License-Identifier: EUPL-1.2

// Extra coverage for the core-agent command surface: the per-command
// error returns in registerApplicationCommands, the hub daemon's
// early-return guards (reached without binding a socket), the
// pollDownload terminal-state loop driven by a stubbed admin endpoint,
// the opencode-models daemon-error branch, and the runCoreAgent binary
// rename path.

package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// --- registerApplicationCommands: per-command error returns ----------

// TestRegisterApplicationCommands_Bad_ConflictPropagates — pre-registering
// each command name with a live Action makes the matching c.Command call
// inside registerApplicationCommands fail, exercising every "!result.OK"
// early-return branch.
func TestRegisterApplicationCommands_Bad_ConflictPropagates(t *testing.T) {
	names := []string{
		"version", "check", "env", "chat", "hub",
		"serve-status", "serve-reload", "serve-profiles",
		"models-download", "models-job", "opencode-models",
	}
	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			c := core.New(core.WithOption("name", "core-agent"))
			// Seed a conflicting executable command so the matching
			// registration inside registerApplicationCommands fails.
			pre := c.Command(name, core.Command{
				Description: "pre-registered conflict",
				Action:      func(_ core.Options) core.Result { return core.Result{OK: true} },
			})
			core.AssertTrue(t, pre.OK)

			r := registerApplicationCommands(c)
			core.AssertFalse(t, r.OK)
		})
	}
}

// --- hub: early-return guards (no socket bind) -----------------------

// TestHub_Bad_NothingToServe — both --no-http and --no-mcp set short-circuits
// to a "nothing to serve" failure after token + audit setup. CORE_WORKSPACE
// points token + audit I/O at a temp dir so nothing lands under $HOME.
func TestHub_Bad_NothingToServe(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	c := newCoreAgent()
	cmds := applicationCommandSet{coreApp: c}

	tokenFile := core.JoinPath(t.TempDir(), "hub.token")
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.hub(core.NewOptions(
			core.Option{Key: "token-file", Value: tokenFile},
			core.Option{Key: "no-http", Value: true},
			core.Option{Key: "no-mcp", Value: true},
		))
	})
	core.AssertFalse(t, r.OK)
	_ = out
	// The token file must have been minted before the guard fired.
	core.AssertTrue(t, c.Fs().IsFile(tokenFile).OK)
}

// TestHub_Bad_TokenFileEmpty — an existing-but-empty token file fails the
// generate-or-load step before any listener is touched.
func TestHub_Bad_TokenFileEmpty(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	c := newCoreAgent()
	cmds := applicationCommandSet{coreApp: c}

	tokenFile := core.JoinPath(t.TempDir(), "hub.token")
	core.AssertTrue(t, c.Fs().Write(tokenFile, "   ").OK)

	r := cmds.hub(core.NewOptions(
		core.Option{Key: "token-file", Value: tokenFile},
		core.Option{Key: "no-mcp", Value: true},
	))
	core.AssertFalse(t, r.OK)
}

// TestHub_Bad_MCPMissingSecret — the MCP plane refuses to start when
// MCP_JWT_SECRET is unset. --no-http keeps the control plane from binding,
// so only the MCP guard is exercised.
func TestHub_Bad_MCPMissingSecret(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("MCP_JWT_SECRET", "")
	c := newCoreAgent()
	cmds := applicationCommandSet{coreApp: c}

	tokenFile := core.JoinPath(t.TempDir(), "hub.token")
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.hub(core.NewOptions(
			core.Option{Key: "token-file", Value: tokenFile},
			core.Option{Key: "no-http", Value: true},
		))
	})
	core.AssertFalse(t, r.OK)
	_ = out
}

// TestHub_Bad_MCPServiceMissing — a Core without the mcp service cannot
// serve the MCP plane. MCP_JWT_SECRET is set so the failure is the missing
// service, not the missing secret.
func TestHub_Bad_MCPServiceMissing(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	t.Setenv("MCP_JWT_SECRET", "test-secret")
	c := core.New(core.WithOption("name", "core-agent"))
	cmds := applicationCommandSet{coreApp: c}

	tokenFile := core.JoinPath(t.TempDir(), "hub.token")
	r := cmds.hub(core.NewOptions(
		core.Option{Key: "token-file", Value: tokenFile},
		core.Option{Key: "no-http", Value: true},
	))
	core.AssertFalse(t, r.OK)
}

// TestHub_Bad_HTTPBuildEngineMissingOpencode — with the HTTP plane enabled
// but no opencode service, buildHubEngine fails and hub returns that error
// before any goroutine serves.
func TestHub_Bad_HTTPBuildEngineMissingOpencode(t *testing.T) {
	t.Setenv("CORE_WORKSPACE", t.TempDir())
	c := core.New(core.WithOption("name", "core-agent"))
	cmds := applicationCommandSet{coreApp: c}

	tokenFile := core.JoinPath(t.TempDir(), "hub.token")
	r := cmds.hub(core.NewOptions(
		core.Option{Key: "token-file", Value: tokenFile},
		core.Option{Key: "no-mcp", Value: true},
	))
	core.AssertFalse(t, r.OK)
}

// --- pollDownload: terminal-state loop --------------------------------

// pollStubServer answers /v1/admin/models/download (the route DownloadJob
// GETs) with the supplied JSON body; an empty body 500s so the poll-error
// branch is exercised.
func pollStubServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/admin/models/download" {
			http.Error(w, "no stub for "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		if body == "" {
			http.Error(w, "stub error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestModels_pollDownload_Good_Done — a job that reports "done" on the first
// poll prints the progress + done lines and returns OK.
func TestModels_pollDownload_Good_Done(t *testing.T) {
	srv := pollStubServer(t, `{
		"job_id": "dl-7",
		"status": "done",
		"progress": 100,
		"bytes": 4096,
		"path": "/Lethean/models/lemer-lite"
	}`)
	admin, ok := buildAdmin(stubAdminOpts(srv.URL))
	core.AssertTrue(t, ok)

	var r core.Result
	out := captureStdout(t, func() {
		r = pollDownload(context.Background(), admin, "dl-7")
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "100%")
	core.AssertContains(t, out, "/Lethean/models/lemer-lite")
}

// TestModels_pollDownload_Bad_Failed — a job that reports "failed" prints the
// failure line carrying the server error and returns non-OK.
func TestModels_pollDownload_Bad_Failed(t *testing.T) {
	srv := pollStubServer(t, `{
		"job_id": "dl-8",
		"status": "failed",
		"progress": 40,
		"error": "upstream allowlist rejected repo"
	}`)
	admin, ok := buildAdmin(stubAdminOpts(srv.URL))
	core.AssertTrue(t, ok)

	var r core.Result
	out := captureStdout(t, func() {
		r = pollDownload(context.Background(), admin, "dl-8")
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "failed")
	core.AssertContains(t, out, "upstream allowlist rejected repo")
}

// TestModels_pollDownload_Bad_PollError — a 500 on the poll route prints the
// poll error and returns non-OK without looping.
func TestModels_pollDownload_Bad_PollError(t *testing.T) {
	srv := pollStubServer(t, "") // route 500s
	admin, ok := buildAdmin(stubAdminOpts(srv.URL))
	core.AssertTrue(t, ok)

	var r core.Result
	out := captureStdout(t, func() {
		r = pollDownload(context.Background(), admin, "dl-9")
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "poll:")
}

// TestModels_modelsDownload_Good_PollsToDone — the full download path: POST
// queues the job, then the poll loop drives it to "done" and returns OK.
func TestModels_modelsDownload_Good_PollsToDone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/admin/models/download" {
			http.Error(w, "no stub", http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		if r.Method == http.MethodPost {
			_, _ = w.Write([]byte(`{"job_id": "dl-10"}`))
			return
		}
		_, _ = w.Write([]byte(`{"job_id": "dl-10", "status": "done", "progress": 100, "path": "/m"}`))
	}))
	t.Cleanup(srv.Close)

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.modelsDownload(stubAdminOpts(srv.URL,
			core.Option{Key: "repo", Value: "lthn/lemer-lite"},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "queued job dl-10")
	core.AssertContains(t, out, "done")
}

// --- opencode-models: daemon-error branch -----------------------------

// TestOpencode_opencodeModels_Bad_DaemonError — when the host has no opencode
// binary, OpencodeHostModels errors and opencodeModels prints the failure and
// returns an empty (non-OK) result.
func TestOpencode_opencodeModels_Bad_DaemonError(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // no opencode on PATH
	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.opencodeModels(core.NewOptions())
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "opencode-models:")
}

// --- chat: open-archive failure --------------------------------------

// TestChat_chat_Bad_OpenArchiveFails — a --workdir whose parent path
// component is an existing file makes chathistory.Open's MkdirAll fail, so
// chat prints the open error and returns non-OK before any session starts.
func TestChat_chat_Bad_OpenArchiveFails(t *testing.T) {
	// Create a regular file, then point the archive under it: PathDir is the
	// file, MkdirAll(<file>) fails, Open fails.
	blocker := core.JoinPath(t.TempDir(), "not-a-dir")
	c := newTestCore(t)
	core.AssertTrue(t, c.Fs().Write(blocker, "x").OK)

	cmds := applicationCommandSet{coreApp: c}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.chat(core.NewOptions(
			core.Option{Key: "user", Value: "owlet"},
			core.Option{Key: "workdir", Value: core.JoinPath(blocker, "chats.duckdb")},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "chat: open archive:")
}

// --- runCoreAgent: binary rename path ---------------------------------

// TestMain_RunCoreAgent_Good_BinaryRename — runCoreAgent builds a fresh core,
// overrides its name + banner with the invoked binary basename, and delegates
// to runApp. A swapped runApp captures the renamed core without standing up
// the full service stack or a CLI run. detectBinaryName reads the real argv[0]
// (the test binary), so the rename target is whatever that basename is — the
// branch is exercised either way; we assert the override took effect.
func TestMain_RunCoreAgent_Good_BinaryRename(t *testing.T) {
	withArgs(t, "core-agent", "version")

	var seenName string
	prevRun := runApp
	runApp = func(c *core.Core, _ []string) error {
		if c != nil {
			seenName = c.App().Name
		}
		return nil
	}
	t.Cleanup(func() { runApp = prevRun })

	err := runCoreAgent()
	core.AssertNoError(t, err)
	// The in-process name was overridden to the invoked binary basename.
	core.AssertEqual(t, detectBinaryName(), seenName)
	core.AssertTrue(t, seenName != "")
}

// TestMain_main_Good_NoError — main delegates to runCoreAgent; a swapped
// runCoreAgent returning nil drives the success path (the if-err false branch)
// without reaching core.Exit.
func TestMain_main_Good_NoError(t *testing.T) {
	prev := runCoreAgent
	runCoreAgent = func() error { return nil }
	t.Cleanup(func() { runCoreAgent = prev })

	main() // must not call core.Exit on the nil-error path
}

// TestMain_RunCoreAgent_Bad_RunAppError — a runApp error propagates out of
// runCoreAgent unchanged.
func TestMain_RunCoreAgent_Bad_RunAppError(t *testing.T) {
	withArgs(t, "core-agent", "version")

	wantErr := core.E("test", "boom", nil)
	prevRun := runApp
	runApp = func(_ *core.Core, _ []string) error { return wantErr }
	t.Cleanup(func() { runApp = prevRun })

	err := runCoreAgent()
	core.AssertError(t, err, wantErr.Error())
}
