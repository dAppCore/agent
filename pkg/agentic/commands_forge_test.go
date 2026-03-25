// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// --- parseForgeArgs ---

func TestCommandsForge_ParseForgeArgs_Good_AllFields(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "myorg"},
		core.Option{Key: "_arg", Value: "myrepo"},
		core.Option{Key: "number", Value: "42"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "myorg", org)
	assert.Equal(t, "myrepo", repo)
	assert.Equal(t, int64(42), num)
}

func TestCommandsForge_ParseForgeArgs_Good_DefaultOrg(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Equal(t, "go-io", repo)
	assert.Equal(t, int64(0), num, "no number provided")
}

func TestCommandsForge_ParseForgeArgs_Bad_EmptyOpts(t *testing.T) {
	opts := core.NewOptions()
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "core", org, "should default to 'core'")
	assert.Empty(t, repo)
	assert.Equal(t, int64(0), num)
}

func TestCommandsForge_ParseForgeArgs_Bad_InvalidNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "repo"},
		core.Option{Key: "number", Value: "not-a-number"},
	)
	_, _, num := parseForgeArgs(opts)
	assert.Equal(t, int64(0), num, "invalid number should parse as 0")
}

// --- fmtIndex ---

func TestCommandsForge_FmtIndex_Good(t *testing.T) {
	assert.Equal(t, "1", fmtIndex(1))
	assert.Equal(t, "42", fmtIndex(42))
	assert.Equal(t, "0", fmtIndex(0))
	assert.Equal(t, "999999", fmtIndex(999999))
}

// --- parseForgeArgs Ugly ---

func TestCommandsForge_ParseForgeArgs_Ugly_OrgSetButNoRepo(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "org", Value: "custom-org"},
	)
	org, repo, num := parseForgeArgs(opts)
	assert.Equal(t, "custom-org", org)
	assert.Empty(t, repo, "repo should be empty when only org is set")
	assert.Equal(t, int64(0), num)
}

func TestCommandsForge_ParseForgeArgs_Ugly_NegativeNumber(t *testing.T) {
	opts := core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "-5"},
	)
	_, _, num := parseForgeArgs(opts)
	assert.Equal(t, int64(-5), num, "negative numbers parse but are semantically invalid")
}

// --- fmtIndex Bad/Ugly ---

func TestCommandsForge_FmtIndex_Bad_Negative(t *testing.T) {
	result := fmtIndex(-1)
	assert.Equal(t, "-1", result, "negative should format as negative string")
}

func TestCommandsForge_FmtIndex_Ugly_VeryLarge(t *testing.T) {
	result := fmtIndex(9999999999)
	assert.Equal(t, "9999999999", result)
}

func TestCommandsForge_FmtIndex_Ugly_MaxInt64(t *testing.T) {
	result := fmtIndex(9223372036854775807) // math.MaxInt64
	assert.NotEmpty(t, result)
	assert.Equal(t, "9223372036854775807", result)
}

// --- Forge commands Ugly (special chars → API returns 404/error) ---

func TestCommandsForge_CmdIssueGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io/<script>"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdIssueList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueList(core.NewOptions(core.Option{Key: "_arg", Value: "repo&evil=true"}))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdIssueComment_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueComment(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
		core.Option{Key: "body", Value: "Hello <b>world</b> & \"quotes\""},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdIssueCreate_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdIssueCreate(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "title", Value: "Fix <b>bug</b> #123"},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdPRGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "../../../etc/passwd"},
		core.Option{Key: "number", Value: "1"},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdPRList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRList(core.NewOptions(core.Option{Key: "_arg", Value: "repo%00null"}))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdPRMerge_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(422) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdPRMerge(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "number", Value: "1"},
		core.Option{Key: "method", Value: "invalid-method"},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdRepoGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoGet(core.NewOptions(
		core.Option{Key: "_arg", Value: "go-io"},
		core.Option{Key: "org", Value: "org/with/slashes"},
	))
	assert.False(t, r.OK)
}

func TestCommandsForge_CmdRepoList_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) }))
	t.Cleanup(srv.Close)
	s, _ := testPrepWithCore(t, srv)
	r := s.cmdRepoList(core.NewOptions(core.Option{Key: "org", Value: "<script>alert(1)</script>"}))
	assert.False(t, r.OK)
}
