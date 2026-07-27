// SPDX-License-Identifier: EUPL-1.2

package lemma

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	core "dappco.re/go"
)

// TestNewAdmin_DefaultHomeTokenPath_Good — empty TokenPath resolves to
// $HOME/Lethean/data/admin.token. Pointing HOME at a temp dir with a
// seeded token exercises the UserHomeDir + JoinPath default branch.
func TestNewAdmin_DefaultHomeTokenPath_Good(t *testing.T) {
	home := t.TempDir()
	// admin.go joins DefaultAdminTokenRelPath = "Lethean/data/admin.token"
	dataDir := filepath.Join(home, "Lethean", "data")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	const tok = "lthn-mlx_homedefault123456"
	if err := writeFile(t, filepath.Join(dataDir, "admin.token"), tok+"\n"); err != nil {
		t.Fatalf("seed token: %v", err)
	}
	t.Setenv("HOME", home)

	admin, err := NewAdmin(AdminConfig{BaseURL: "http://127.0.0.1:0"})
	core.AssertTrue(t, err == nil, "NewAdmin with default home token path should succeed")
	core.AssertTrue(t, admin != nil, "admin handle should be non-nil")
}

// TestNewAdmin_DefaultHomeTokenMissing_Bad — default path with no token
// file present surfaces the load-token error (the home-dir miss branch).
func TestNewAdmin_DefaultHomeTokenMissing_Bad(t *testing.T) {
	home := t.TempDir() // empty: no Lethean/data/admin.token
	t.Setenv("HOME", home)

	_, err := NewAdmin(AdminConfig{})
	core.AssertTrue(t, err != nil, "missing default token file should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "admin token"), "error should mention admin token: "+errStr(err))
}

// TestLoadTokenFromFile_ReadFail_Bad — a path that does not exist makes
// loadTokenFromFile return the read-failure error (the !r.OK branch).
func TestLoadTokenFromFile_ReadFail_Bad(t *testing.T) {
	_, err := loadTokenFromFile(filepath.Join(t.TempDir(), "does-not-exist.token"))
	core.AssertTrue(t, err != nil, "reading a missing token file should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "read"), "error should mention read: "+errStr(err))
}

// TestLoadTokenFromFile_Good — a seeded, padded token reads back trimmed.
func TestLoadTokenFromFile_Good(t *testing.T) {
	p := filepath.Join(t.TempDir(), "admin.token")
	if err := writeFile(t, p, "  lthn-mlx_trimmed_me  \n"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	tok, err := loadTokenFromFile(p)
	core.AssertTrue(t, err == nil, "loadTokenFromFile should succeed")
	core.AssertEqual(t, "lthn-mlx_trimmed_me", tok)
}

// TestAdminStatus_ServerError_Bad — Status wraps a 5xx from the daemon
// into an error (the doJSON status>=400 + Status error-wrap branches).
func TestAdminStatus_ServerError_Bad(t *testing.T) {
	const tok = "tok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "serve not loaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: tok})
	_, err := admin.Status(context.Background())
	core.AssertTrue(t, err != nil, "Status against a 503 should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "503"), "error should carry the 503: "+errStr(err))
}

// TestAdminProfiles_ServerError_Bad — Profiles surfaces a 5xx as error.
func TestAdminProfiles_ServerError_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "profiles dir unreadable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: "tok"})
	_, err := admin.Profiles(context.Background())
	core.AssertTrue(t, err != nil, "Profiles against a 500 should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "500"), "error should carry the 500: "+errStr(err))
}

// TestAdminReload_ServerError_Bad — Reload with a valid confirm_machine
// still surfaces a server rejection (the post-flight doJSON error wrap,
// distinct from the pre-flight confirm-required guard).
func TestAdminReload_ServerError_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "machine hash mismatch", http.StatusConflict)
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: "tok"})
	err := admin.Reload(context.Background(), ReloadRequest{ConfirmMachine: "some-hash"})
	core.AssertTrue(t, err != nil, "Reload rejected by server should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "409"), "error should carry the 409: "+errStr(err))
}

// TestAdminDownload_MissingRepoID_Bad — empty repo_id is rejected
// pre-flight, before any HTTP (the Trim guard).
func TestAdminDownload_MissingRepoID_Bad(t *testing.T) {
	admin, _ := NewAdmin(AdminConfig{BaseURL: "http://127.0.0.1:0", Token: "tok"})
	_, err := admin.Download(context.Background(), DownloadRequest{RepoID: "   "})
	core.AssertTrue(t, err != nil, "blank repo_id should error pre-flight")
	core.AssertTrue(t, strings.Contains(err.Error(), "repo_id"), "error should mention repo_id: "+errStr(err))
}

// TestAdminDownload_ServerOmitsJobID_Bad — a 200 response that decodes
// fine but carries no job_id is rejected (the empty-job_id guard).
func TestAdminDownload_ServerOmitsJobID_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		// Valid JSON, status accepted, but no job_id field.
		_, _ = w.Write([]byte(`{"status":"pending","repo_id":"lthn/lemer-lite"}`))
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: "tok"})
	_, err := admin.Download(context.Background(), DownloadRequest{RepoID: "lthn/lemer-lite"})
	core.AssertTrue(t, err != nil, "missing job_id in response should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "job_id"), "error should mention job_id: "+errStr(err))
}

// TestAdminDownloadJob_MissingJobID_Bad — empty job id is rejected
// pre-flight (the Trim guard before the HTTP call).
func TestAdminDownloadJob_MissingJobID_Bad(t *testing.T) {
	admin, _ := NewAdmin(AdminConfig{BaseURL: "http://127.0.0.1:0", Token: "tok"})
	_, err := admin.DownloadJob(context.Background(), "  ")
	core.AssertTrue(t, err != nil, "blank job id should error pre-flight")
	core.AssertTrue(t, strings.Contains(err.Error(), "job id"), "error should mention job id: "+errStr(err))
}

// TestAdminDownloadJob_ServerError_Bad — DownloadJob surfaces a 5xx from
// the daemon (the DownloadJob error-wrap branch with a real job id set).
func TestAdminDownloadJob_ServerError_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no such job", http.StatusNotFound)
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: "tok"})
	_, err := admin.DownloadJob(context.Background(), "dl-job-unknown")
	core.AssertTrue(t, err != nil, "DownloadJob for a 404 should error")
	core.AssertTrue(t, strings.Contains(err.Error(), "404"), "error should carry the 404: "+errStr(err))
}

// TestAdminDoJSON_DecodeError_Bad — a 200 with a body that does not
// match the target shape surfaces the decode-response error branch.
func TestAdminDoJSON_DecodeError_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "application/json")
		// "config" is an int where ServeStatusConfig (an object) is wanted.
		_, _ = w.Write([]byte(`{"model_path":"/m","config":12345}`))
	}))
	defer srv.Close()

	admin, _ := NewAdmin(AdminConfig{BaseURL: srv.URL, Token: "tok"})
	_, err := admin.Status(context.Background())
	core.AssertTrue(t, err != nil, "malformed JSON shape should error on decode")
	core.AssertTrue(t, strings.Contains(err.Error(), "decode"), "error should mention decode: "+errStr(err))
}

// TestAdminDoJSON_TransportError_Bad — pointing the client at a closed
// listener triggers the transport error branch of doJSON.
func TestAdminDoJSON_TransportError_Bad(t *testing.T) {
	// Stand a server up, capture its URL, then close it so the dial fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	admin, _ := NewAdmin(AdminConfig{
		BaseURL: closedURL,
		Token:   "tok",
		Timeout: 500 * time.Millisecond,
	})
	_, err := admin.Machine(context.Background())
	core.AssertTrue(t, err != nil, "request to a closed listener should error at transport")
}

// errStr renders an error for assertion messages without tripping the
// nil-deref when an assertion already proved err non-nil.
func errStr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}
