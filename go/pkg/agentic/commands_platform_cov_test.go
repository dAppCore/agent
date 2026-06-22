// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestCommandsPlatformCov_CmdSyncPush_Ugly_PushError overrides the injectable
// push seam to fail, exercising the !result.OK arm of cmdSyncPush.
func TestCommandsPlatformCov_CmdSyncPush_Ugly_PushError(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	original := syncPushInput
	t.Cleanup(func() { syncPushInput = original })
	syncPushInput = func(_ *PrepSubsystem, _ context.Context, _ SyncPushInput) (SyncPushOutput, error) {
		return SyncPushOutput{}, core.E("agentic.syncPush", "remote push failed", nil)
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSyncPush(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "remote push failed")
	core.AssertContains(t, output, "error:")
}

// TestCommandsPlatformCov_CmdSyncPull_Ugly_PullError overrides the injectable
// pull seam to fail, exercising the !result.OK arm of cmdSyncPull.
func TestCommandsPlatformCov_CmdSyncPull_Ugly_PullError(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	original := syncPullInput
	t.Cleanup(func() { syncPullInput = original })
	syncPullInput = func(_ *PrepSubsystem, _ context.Context, _ SyncPullInput) (SyncPullOutput, error) {
		return SyncPullOutput{}, core.E("agentic.syncPull", "remote pull failed", nil)
	}

	var r core.Result
	output := captureStdout(t, func() { r = s.cmdSyncPull(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "remote pull failed")
	core.AssertContains(t, output, "error:")
}

// TestCommandsPlatformCov_CmdAuthRevoke_Bad_MissingKeyID — no key-id argument
// prints usage and returns the required-field error before any HTTP call.
func TestCommandsPlatformCov_CmdAuthRevoke_Bad_MissingKeyID(t *testing.T) {
	s := testPrepWithPlatformServer(t, nil, "secret-token")
	var r core.Result
	output := captureStdout(t, func() { r = s.cmdAuthRevoke(core.NewOptions()) })
	core.AssertFalse(t, r.OK)
	core.AssertContains(t, r.Value.(error).Error(), "key_id is required")
	core.AssertContains(t, output, "usage: core-agent auth revoke")
}

// TestCommandsPlatformCov_CmdCreditsHistory_Good_EmptyList — a backend returning
// zero entries prints the "no credit entries" line and returns OK.
func TestCommandsPlatformCov_CmdCreditsHistory_Good_EmptyList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"entries":[],"total":0}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdCreditsHistory(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, output, "no credit entries")
}

// TestCommandsPlatformCov_CmdCreditsHistory_Good_PopulatedRows — a populated
// history renders each entry row and the total.
func TestCommandsPlatformCov_CmdCreditsHistory_Good_PopulatedRows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertContains(t, r.URL.Path, "/credits/")
		_, _ = w.Write([]byte(`{"data":{"entries":[{"id":1,"task_type":"fleet-task","amount":2,"balance_after":12},{"id":2,"task_type":"review","amount":-1,"balance_after":11}],"total":2}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdCreditsHistory(core.NewOptions(core.Option{Key: "_arg", Value: "charon"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, output, "fleet-task")
	core.AssertContains(t, output, "review")
	core.AssertContains(t, output, "total: 2")
}

// TestCommandsPlatformCov_CmdAuthProvision_Good_AllOptionalFields exercises the
// permissions / ip-restrictions / expires optional print lines on success.
func TestCommandsPlatformCov_CmdAuthProvision_Good_AllOptionalFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":9,"name":"codex","prefix":"ck_abc","key":"ck_abc_secret","permissions":["plans:read","plans:write"],"ip_restrictions":["10.0.0.0/8"],"expires_at":"2026-12-01T00:00:00Z"}}`))
	}))
	defer srv.Close()
	s := testPrepWithPlatformServer(t, srv, "secret-token")

	var r core.Result
	output := captureStdout(t, func() {
		r = s.cmdAuthProvision(core.NewOptions(core.Option{Key: "_arg", Value: "oauth-user-9"}))
	})
	core.AssertTrue(t, r.OK)
	core.AssertContains(t, output, "key id:      9")
	core.AssertContains(t, output, "key:         ck_abc_secret")
	core.AssertContains(t, output, "permissions: plans:read,plans:write")
	core.AssertContains(t, output, "ip restrictions: 10.0.0.0/8")
	core.AssertContains(t, output, "expires:     2026-12-01T00:00:00Z")
}

// TestCommandsPlatformCov_RegisterPlatformCommands_Ugly_DuplicateConflict — a
// second registration of the platform commands fails on the first duplicate,
// exercising the early-return guard in the registrar.
func TestCommandsPlatformCov_RegisterPlatformCommands_Ugly_DuplicateConflict(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	first := s.registerPlatformCommands()
	core.RequireTrue(t, first.OK)

	second := s.registerPlatformCommands()
	core.AssertFalse(t, second.OK)
}
