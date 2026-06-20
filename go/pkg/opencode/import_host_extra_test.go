// SPDX-Licence-Identifier: EUPL-1.2

package opencode

import (
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

// TestImportHost_readHostAuthJSON_GoodBad — a missing auth.json yields an empty
// map; a present one is parsed into the provider map.
func TestImportHost_readHostAuthJSON_GoodBad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	core.AssertEqual(t, 0, len(readHostAuthJSON()))

	dir := core.PathJoin(home, ".local/share/opencode")
	core.AssertTrue(t, core.MkdirAll(dir, 0o755).OK)
	core.AssertTrue(t, core.WriteFile(core.PathJoin(dir, "auth.json"), []byte(`{"anthropic":{"type":"api"}}`), 0o600).OK)
	got := readHostAuthJSON()
	core.AssertEqual(t, 1, len(got))
}

// TestImportHost_persistProjects_Empty — an empty project array writes nothing
// and returns a zero count.
func TestImportHost_persistProjects_Empty(t *testing.T) {
	c := core.New(core.WithOption("name", "opencode-test"))
	core.AssertEqual(t, 0, persistProjects(c, []any{}, core.Now()))
}

// TestImportHost_stringFrom_Good — extract a string value; non-string or
// missing keys yield "".
func TestImportHost_stringFrom_Good(t *testing.T) {
	core.AssertEqual(t, "v", stringFrom(map[string]any{"k": "v"}, "k"))
	core.AssertEqual(t, "", stringFrom(map[string]any{"k": 1}, "k"))
	core.AssertEqual(t, "", stringFrom(map[string]any{}, "missing"))
}

// TestImportHost_projectNameFrom_Good — empty/"/" worktree falls back to the
// source id; a real path yields its basename.
func TestImportHost_projectNameFrom_Good(t *testing.T) {
	core.AssertEqual(t, "fb", projectNameFrom("", "fb"))
	core.AssertEqual(t, "fb", projectNameFrom("/", "fb"))
	core.AssertEqual(t, "repo", projectNameFrom("/home/user/repo", "fb"))
}

// TestImportHost_importFetchJSON_Good — a 200 JSON body decodes to the
// parsed shape and the Authorization header is forwarded.
func TestImportHost_importFetchJSON_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "Basic cred", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"p1"}]`))
	}))
	defer srv.Close()

	got, err := importFetchJSON(srv.URL+"/project", "Basic cred")
	core.AssertNoError(t, err)
	arr, ok := got.([]any)
	core.AssertTrue(t, ok)
	core.AssertEqual(t, 1, len(arr))
}

// TestImportHost_importFetchJSON_Bad_4xx — a 4xx surfaces the status code
// + body as an error (the >=400 leg).
func TestImportHost_importFetchJSON_Bad_4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("denied"))
	}))
	defer srv.Close()

	_, err := importFetchJSON(srv.URL, "")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "HTTP 403")
}

// TestImportHost_importFetchJSON_Bad_Decode — a 200 with non-JSON body
// fails the decode leg.
func TestImportHost_importFetchJSON_Bad_Decode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	_, err := importFetchJSON(srv.URL, "")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "decode")
}

// TestImportHost_importFetchJSON_Bad_RequestBuild — a malformed URL fails
// the request-build leg before any network call.
func TestImportHost_importFetchJSON_Bad_RequestBuild(t *testing.T) {
	_, err := importFetchJSON("://bad", "")
	core.AssertError(t, err)
}

// TestImportHost_importFetchJSON_Bad_Unreachable — a dead target fails the
// client.Do leg.
func TestImportHost_importFetchJSON_Bad_Unreachable(t *testing.T) {
	_, err := importFetchJSON("http://127.0.0.1:1", "")
	core.AssertError(t, err)
}
