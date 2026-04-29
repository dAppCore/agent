// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	core "dappco.re/go"
)

func TestPipelineCommands_RegisterPipelineCommands_Good_Case(t *testing.T) {
	s, c := testPrepWithCore(t, nil)

	s.registerPipelineCommands()

	commands := c.Commands()
	core.AssertContains(t, commands, "pipeline")
	core.AssertContains(t, commands, "pipeline/audit")
	core.AssertContains(t, commands, "pipeline/epic")
	core.AssertContains(t, commands, "pipeline/epic/create")
	core.AssertContains(t, commands, "pipeline/epic/run")
	core.AssertContains(t, commands, "pipeline/epic/status")
	core.AssertContains(t, commands, "pipeline/epic/sync")
	core.AssertContains(t, commands, "pipeline/monitor")
	core.AssertContains(t, commands, "pipeline/fix")
	core.AssertContains(t, commands, "pipeline/fix/reviews")
	core.AssertContains(t, commands, "pipeline/fix/conflicts")
	core.AssertContains(t, commands, "pipeline/fix/format")
	core.AssertContains(t, commands, "pipeline/fix/threads")
	core.AssertContains(t, commands, "pipeline/onboard")
	core.AssertContains(t, commands, "pipeline/budget")
	core.AssertContains(t, commands, "pipeline/budget/plan")
	core.AssertContains(t, commands, "pipeline/budget/log")
	core.AssertContains(t, commands, "pipeline/training")
	core.AssertContains(t, commands, "pipeline/training/capture")
	core.AssertContains(t, commands, "pipeline/training/stats")
	core.AssertContains(t, commands, "pipeline/training/export")
}

func TestPipelineCommands_CmdPipeline_Good_Help(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipeline(core.NewOptions())
		core.RequireTrue(t, result.OK)
	})

	core.AssertContains(t, output, "core-agent pipeline/audit <repo>")
	core.AssertContains(t, output, "core-agent pipeline/epic/create <repo>")
	core.AssertContains(t, output, "core-agent pipeline/onboard <repo>")
}

func TestPipelineCommands_CmdPipeline_Bad_UnknownCommand(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipeline(core.NewOptions(core.Option{Key: "_arg", Value: "unknown"}))

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "unknown pipeline command")
}

type pipelineTestIssue struct {
	Number int
	Title  string
	Body   string
	State  string
	Labels []string
}

type pipelineTestPR struct {
	Number                  int
	Title                   string
	State                   string
	Merged                  bool
	Mergeable               *bool
	MergeableState          string
	HeadRef                 string
	HeadSHA                 string
	BaseRef                 string
	ReviewThreadsTotal      int
	ReviewThreadsResolved   int
	ReviewThreadsUnresolved int
	ReviewComments          int
	Comments                int
	Reactions               map[string]int
	Statuses                []map[string]any
}

type pipelineTestRepo struct {
	DefaultBranch string
	RefSHA        string
	Labels        map[string]int64
	Issues        map[int]*pipelineTestIssue
	Pulls         map[int]*pipelineTestPR
	Comments      map[int][]string
	Branches      map[string]string
	Merged        []int
}

func newPipelineTestRepo() *pipelineTestRepo {
	return &pipelineTestRepo{
		DefaultBranch: "dev",
		RefSHA:        "deadbeef",
		Labels: map[string]int64{
			"agentic":     1,
			"audit":       2,
			"security":    3,
			"quality":     4,
			"testing":     5,
			"performance": 6,
			"docs":        7,
			"epic":        8,
			"critical":    9,
			"high":        10,
			"medium":      11,
			"low":         12,
		},
		Issues:   map[int]*pipelineTestIssue{},
		Pulls:    map[int]*pipelineTestPR{},
		Comments: map[int][]string{},
		Branches: map[string]string{"dev": "deadbeef"},
		Merged:   []int{},
	}
}

func newPipelineTestServer(t *testing.T, repos map[string]*pipelineTestRepo) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		parts := core.Split(path, "/")

		if len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "orgs" && parts[4] == "repos" && r.Method == http.MethodGet {
			org := parts[3]
			_ = org
			names := make([]string, 0, len(repos))
			for name := range repos {
				names = append(names, name)
			}
			sort.Strings(names)
			out := []map[string]any{}
			for _, name := range names {
				repo := repos[name]
				out = append(out, map[string]any{
					"name":           name,
					"default_branch": repo.DefaultBranch,
					"html_url":       "https://forge.test/core/" + name,
				})
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(core.JSONMarshalString(out)))
			return
		}

		if len(parts) < 5 || parts[0] != "api" || parts[1] != "v1" || parts[2] != "repos" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		repoName := parts[4]
		repo, ok := repos[repoName]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if len(parts) == 5 && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
				"name":           repoName,
				"default_branch": repo.DefaultBranch,
				"html_url":       "https://forge.test/core/" + repoName,
			})))
			return
		}

		if len(parts) >= 6 && parts[5] == "labels" {
			switch r.Method {
			case http.MethodGet:
				names := make([]string, 0, len(repo.Labels))
				for name := range repo.Labels {
					names = append(names, name)
				}
				sort.Strings(names)
				out := []map[string]any{}
				for _, name := range names {
					out = append(out, map[string]any{"id": repo.Labels[name], "name": name})
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(out)))
				return
			case http.MethodPost:
				bodyResult := core.ReadAll(r.Body)
				core.RequireTrue(t, bodyResult.OK)
				var payload map[string]any
				core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
				name := payload["name"].(string)
				repo.Labels[name] = int64(len(repo.Labels) + 1)
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"id": repo.Labels[name]})))
				return
			}
		}

		if len(parts) >= 6 && parts[5] == "issues" {
			switch {
			case len(parts) == 6 && r.Method == http.MethodGet:
				numbers := make([]int, 0, len(repo.Issues))
				for number := range repo.Issues {
					numbers = append(numbers, number)
				}
				sort.Ints(numbers)
				out := []map[string]any{}
				for _, number := range numbers {
					issue := repo.Issues[number]
					if issue == nil {
						continue
					}
					if state := r.URL.Query().Get("state"); state == "open" && issue.State == "closed" {
						continue
					}
					out = append(out, pipelineIssuePayload(repoName, issue))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(out)))
				return
			case len(parts) == 6 && r.Method == http.MethodPost:
				bodyResult := core.ReadAll(r.Body)
				core.RequireTrue(t, bodyResult.OK)
				var payload map[string]any
				core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
				next := 1
				for number := range repo.Issues {
					if number >= next {
						next = number + 1
					}
				}
				title := payload["title"].(string)
				body, _ := payload["body"].(string)
				labels := []string{}
				if raw, ok := payload["labels"].([]any); ok {
					for _, labelID := range raw {
						for name, id := range repo.Labels {
							if int64(labelID.(float64)) == id {
								labels = append(labels, name)
							}
						}
					}
				}
				repo.Issues[next] = &pipelineTestIssue{Number: next, Title: title, Body: body, State: "open", Labels: labels}
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{
					"number":   next,
					"html_url": "https://forge.test/core/" + repoName + "/issues/" + core.Sprint(next),
				})))
				return
			case len(parts) == 7 && r.Method == http.MethodGet:
				number := parseIntString(parts[6])
				issue := repo.Issues[number]
				if issue == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(pipelineIssuePayload(repoName, issue))))
				return
			case len(parts) == 7 && r.Method == http.MethodPatch:
				number := parseIntString(parts[6])
				issue := repo.Issues[number]
				if issue == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				bodyResult := core.ReadAll(r.Body)
				core.RequireTrue(t, bodyResult.OK)
				var payload map[string]any
				core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
				if state, ok := payload["state"].(string); ok {
					issue.State = state
				}
				if body, ok := payload["body"].(string); ok {
					issue.Body = body
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(pipelineIssuePayload(repoName, issue))))
				return
			case len(parts) == 8 && parts[7] == "comments" && r.Method == http.MethodPost:
				number := parseIntString(parts[6])
				bodyResult := core.ReadAll(r.Body)
				core.RequireTrue(t, bodyResult.OK)
				var payload map[string]any
				core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
				repo.Comments[number] = append(repo.Comments[number], payload["body"].(string))
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"id": len(repo.Comments[number])})))
				return
			}
		}

		if len(parts) >= 6 && parts[5] == "pulls" {
			switch {
			case len(parts) == 6 && r.Method == http.MethodGet:
				numbers := make([]int, 0, len(repo.Pulls))
				for number := range repo.Pulls {
					numbers = append(numbers, number)
				}
				sort.Ints(numbers)
				out := []map[string]any{}
				for _, number := range numbers {
					pullRequest := repo.Pulls[number]
					if pullRequest == nil {
						continue
					}
					if state := r.URL.Query().Get("state"); state == "open" && pullRequest.State != "open" {
						continue
					}
					out = append(out, pipelinePRPayload(repoName, pullRequest))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(out)))
				return
			case len(parts) == 7 && r.Method == http.MethodGet:
				number := parseIntString(parts[6])
				pullRequest := repo.Pulls[number]
				if pullRequest == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(pipelinePRPayload(repoName, pullRequest))))
				return
			case len(parts) == 8 && parts[7] == "merge" && r.Method == http.MethodPost:
				number := parseIntString(parts[6])
				repo.Merged = append(repo.Merged, number)
				if pullRequest := repo.Pulls[number]; pullRequest != nil {
					pullRequest.State = "merged"
					pullRequest.Merged = true
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{}"))
				return
			}
		}

		if len(parts) == 8 && parts[5] == "commits" && parts[7] == "status" && r.Method == http.MethodGet {
			sha := parts[6]
			for _, pullRequest := range repo.Pulls {
				if pullRequest != nil && pullRequest.HeadSHA == sha {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"statuses": pullRequest.Statuses})))
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"statuses": []map[string]any{}})))
			return
		}

		if len(parts) >= 7 && parts[5] == "git" && parts[6] == "refs" {
			switch {
			case len(parts) >= 9 && parts[7] == "heads" && r.Method == http.MethodGet:
				branch := core.Join("/", parts[8:]...)
				sha := repo.Branches[branch]
				if sha == "" {
					sha = repo.RefSHA
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"object": map[string]any{"sha": sha}})))
				return
			case len(parts) == 7 && r.Method == http.MethodPost:
				bodyResult := core.ReadAll(r.Body)
				core.RequireTrue(t, bodyResult.OK)
				var payload map[string]any
				core.RequireTrue(t, core.JSONUnmarshalString(bodyResult.Value.(string), &payload).OK)
				ref := payload["ref"].(string)
				sha := payload["sha"].(string)
				branch := ref[len("refs/heads/"):]
				repo.Branches[branch] = sha
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(core.JSONMarshalString(map[string]any{"ref": ref})))
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	t.Cleanup(server.Close)
	return server
}

func pipelineIssuePayload(repoName string, issue *pipelineTestIssue) map[string]any {
	labels := []map[string]any{}
	for _, label := range issue.Labels {
		labels = append(labels, map[string]any{"name": label})
	}
	return map[string]any{
		"number":   issue.Number,
		"title":    issue.Title,
		"body":     issue.Body,
		"state":    issue.State,
		"labels":   labels,
		"html_url": "https://forge.test/core/" + repoName + "/issues/" + core.Sprint(issue.Number),
	}
}

func pipelinePRPayload(repoName string, pullRequest *pipelineTestPR) map[string]any {
	payload := map[string]any{
		"number":                    pullRequest.Number,
		"title":                     pullRequest.Title,
		"state":                     pullRequest.State,
		"html_url":                  "https://forge.test/core/" + repoName + "/pulls/" + core.Sprint(pullRequest.Number),
		"merged":                    pullRequest.Merged,
		"mergeable":                 pullRequest.Mergeable,
		"mergeable_state":           pullRequest.MergeableState,
		"review_threads_total":      pullRequest.ReviewThreadsTotal,
		"review_threads_resolved":   pullRequest.ReviewThreadsResolved,
		"review_threads_unresolved": pullRequest.ReviewThreadsUnresolved,
		"review_comments":           pullRequest.ReviewComments,
		"comments":                  pullRequest.Comments,
		"reactions":                 pullRequest.Reactions,
		"head": map[string]any{
			"ref": pullRequest.HeadRef,
			"sha": pullRequest.HeadSHA,
			"repo": map[string]any{
				"updated_at": "2026-04-25T12:00:00Z",
				"pushed_at":  "2026-04-25T12:00:00Z",
			},
		},
		"base": map[string]any{
			"ref": pullRequest.BaseRef,
		},
	}
	if pullRequest.Mergeable == nil {
		payload["mergeable"] = nil
	}
	return payload
}

func boolPtr(value bool) *bool {
	return &value
}
