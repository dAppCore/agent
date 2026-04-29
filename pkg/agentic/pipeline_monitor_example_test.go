// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
)

func pipelineExampleReader(serverURL string) *pipelineForgeMetaReader {
	return &pipelineForgeMetaReader{
		subsystem: &PrepSubsystem{
			forgeURL:   serverURL,
			forgeToken: "test-token",
		},
		org: "core",
	}
}

func ExampleForgeMetaReader_GetPRMeta() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/pulls/12":
			_, _ = w.Write([]byte(`{"number":12,"state":"open","mergeable":true,"mergeable_state":"clean","head":{"ref":"agent/fix","sha":"sha-12"},"base":{"ref":"dev"}}`))
		case "/api/v1/repos/core/go-io/commits/sha-12/status":
			_, _ = w.Write([]byte(`{"statuses":[{"context":"qa","status":"success"}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	meta, err := pipelineExampleReader(srv.URL).GetPRMeta(context.Background(), "go-io", 12)
	core.Println(err == nil)
	core.Println(meta.Mergeable)
	core.Println(len(meta.Checks))
	// Output:
	// true
	// mergeable
	// 1
}

func ExampleForgeMetaReader_GetEpicMeta() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/repos/core/go-io/issues/1":
			_, _ = w.Write([]byte(`{"number":1,"title":"Epic","state":"open","body":"Epic branch: ` + "`agent/epic`" + `\n- [x] #2 Done child"}`))
		case "/api/v1/repos/core/go-io/issues/2":
			_, _ = w.Write([]byte(`{"number":2,"title":"Done child","state":"closed"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	meta, err := pipelineExampleReader(srv.URL).GetEpicMeta(context.Background(), "go-io", 1)
	core.Println(err == nil)
	core.Println(meta.Branch)
	core.Println(len(meta.Children))
	// Output:
	// true
	// agent/epic
	// 1
}

func ExampleForgeMetaReader_GetIssueState() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/core/go-io/issues/7" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`{"number":7,"title":"Fix flaky tests","state":"closed","labels":[{"name":"bug"}]}`))
	}))
	defer srv.Close()

	state, err := pipelineExampleReader(srv.URL).GetIssueState(context.Background(), "go-io", 7)
	core.Println(err == nil)
	core.Println(state.State)
	core.Println(state.Labels[0])
	// Output:
	// true
	// closed
	// bug
}

func ExampleForgeMetaReader_GetCommentReactions() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/core/go-io/issues/comments/55/reactions" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(`[{"content":"eyes"},{"content":"eyes"},{"content":"rocket"}]`))
	}))
	defer srv.Close()

	reactions, err := pipelineExampleReader(srv.URL).GetCommentReactions(context.Background(), "go-io", 55)
	counts := map[string]int{}
	for _, reaction := range reactions {
		counts[reaction.Content] = reaction.Count
	}
	core.Println(err == nil)
	core.Println(counts["eyes"])
	core.Println(counts["rocket"])
	// Output:
	// true
	// 2
	// 1
}
