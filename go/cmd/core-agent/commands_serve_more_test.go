// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// adminStubServer returns an httptest server that answers the lthn-mlx
// /v1/admin/* routes serveStatus / serveReload / serveProfiles hit, with
// the JSON bodies (and status codes) supplied per-path. A path absent from
// routes answers 500 so the handler's error branch is exercised.
func adminStubServer(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok {
			http.Error(w, "no stub for "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
}

// stubAdminOpts builds the options buildAdmin needs to reach a stub server:
// an explicit token (so NewAdmin skips the on-disk token load) plus the
// stub base URL.
func stubAdminOpts(baseURL string, extra ...core.Option) core.Options {
	opts := []core.Option{
		{Key: "admin-token", Value: "test-token"},
		{Key: "base-url", Value: baseURL},
	}
	return core.NewOptions(append(opts, extra...)...)
}

// TestServe_serveStatus_Good_FullSnapshot — a populated status JSON renders
// every optional line (profile, batch, adapter) and returns OK.
func TestServe_serveStatus_Good_FullSnapshot(t *testing.T) {
	srv := adminStubServer(t, map[string]string{
		"/v1/admin/serve/status": `{
			"model_path": "/Lethean/models/lemer-lite",
			"profile_path": "/Lethean/profiles/fast.json",
			"runtime": "lthn-mlx",
			"loaded_at_unix": 1700000000,
			"config": {
				"context_length": 8192,
				"parallel_slots": 4,
				"prompt_cache": true,
				"cache_policy": "lru",
				"cache_mode": "prompt",
				"batch_size": 512,
				"prefill_chunk_size": 128,
				"adapter_path": "/Lethean/adapters/lek.safetensors"
			}
		}`,
	})
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveStatus(stubAdminOpts(srv.URL)) })

	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "/Lethean/models/lemer-lite")
	core.AssertContains(t, out, "/Lethean/profiles/fast.json")
	core.AssertContains(t, out, "lthn-mlx")
	core.AssertContains(t, out, "8192")
	core.AssertContains(t, out, "prefill chunk 128")
	core.AssertContains(t, out, "/Lethean/adapters/lek.safetensors")
}

// TestServe_serveStatus_Good_MinimalSnapshot — a status JSON with no
// profile/batch/adapter omits those optional lines but still returns OK.
func TestServe_serveStatus_Good_MinimalSnapshot(t *testing.T) {
	srv := adminStubServer(t, map[string]string{
		"/v1/admin/serve/status": `{
			"model_path": "/m",
			"runtime": "lthn-mlx",
			"loaded_at_unix": 1700000000,
			"config": {"context_length": 4096, "prompt_cache": false}
		}`,
	})
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveStatus(stubAdminOpts(srv.URL)) })

	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "4096")
	core.AssertFalse(t, core.Contains(out, "adapter:"))
}

// TestServe_serveStatus_Bad_DaemonError — a reachable admin client whose
// daemon answers 500 prints the error and returns non-OK.
func TestServe_serveStatus_Bad_DaemonError(t *testing.T) {
	srv := adminStubServer(t, map[string]string{}) // every path 500s
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveStatus(stubAdminOpts(srv.URL)) })

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "serve-status:")
}

// TestServe_serveReload_Bad_NothingToDo — --confirm with no model/profile/
// context is the "nothing to do" guard, non-OK without touching the daemon.
func TestServe_serveReload_Bad_NothingToDo(t *testing.T) {
	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.serveReload(core.NewOptions(core.Option{Key: "confirm", Value: "abc"}))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "nothing to do")
}

// TestServe_serveReload_Good_Reloads — confirm + model + a stub that 200s the
// reload route returns OK.
func TestServe_serveReload_Good_Reloads(t *testing.T) {
	srv := adminStubServer(t, map[string]string{
		"/v1/admin/serve/reload": `{}`,
	})
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.serveReload(stubAdminOpts(srv.URL,
			core.Option{Key: "confirm", Value: "machine-hash"},
			core.Option{Key: "model", Value: "/Lethean/models/lemer-lite"},
			core.Option{Key: "context", Value: 8192},
		))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "serve-reload: ok")
}

// TestServe_serveReload_Bad_DaemonError — confirm + model but the reload route
// 500s prints the error and returns non-OK.
func TestServe_serveReload_Bad_DaemonError(t *testing.T) {
	srv := adminStubServer(t, map[string]string{}) // reload path 500s
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() {
		r = cmds.serveReload(stubAdminOpts(srv.URL,
			core.Option{Key: "confirm", Value: "machine-hash"},
			core.Option{Key: "model", Value: "/m"},
		))
	})
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "serve-reload:")
}

// TestServe_serveProfiles_Good_List — a profiles list renders each entry and
// returns OK.
func TestServe_serveProfiles_Good_List(t *testing.T) {
	srv := adminStubServer(t, map[string]string{
		"/v1/admin/profiles": `{
			"dir": "/Lethean/profiles",
			"profiles": [
				{"name": "fast", "backend": "mlx", "model": "lemer-lite"},
				{"name": "quality", "backend": "mlx", "model": "lemer-31b"}
			]
		}`,
	})
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveProfiles(stubAdminOpts(srv.URL)) })

	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "/Lethean/profiles")
	core.AssertContains(t, out, "fast")
	core.AssertContains(t, out, "quality")
}

// TestServe_serveProfiles_Good_Empty — an empty profiles list takes the
// "(none)" branch and still returns OK.
func TestServe_serveProfiles_Good_Empty(t *testing.T) {
	srv := adminStubServer(t, map[string]string{
		"/v1/admin/profiles": `{"dir": "/Lethean/profiles", "profiles": []}`,
	})
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveProfiles(stubAdminOpts(srv.URL)) })

	core.AssertTrue(t, r.OK)
	core.AssertContains(t, out, "(none)")
}

// TestServe_serveProfiles_Bad_DaemonError — a 500 on the profiles route prints
// the error and returns non-OK.
func TestServe_serveProfiles_Bad_DaemonError(t *testing.T) {
	srv := adminStubServer(t, map[string]string{}) // profiles path 500s
	defer srv.Close()

	cmds := applicationCommandSet{coreApp: newTestCore(t)}
	var r core.Result
	out := captureStdout(t, func() { r = cmds.serveProfiles(stubAdminOpts(srv.URL)) })

	core.AssertFalse(t, r.OK)
	core.AssertContains(t, out, "serve-profiles:")
}
