// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	// Note: AX-6 — this wrapper owns the direct Forge HTTP boundary so it can preserve response status semantics.
	"net/http"

	core "dappco.re/go"
)

type forgeClient struct {
	baseURL string
	token   string
}

type forgeAPIError struct {
	StatusCode int
	Path       string
	Message    string
}

type Repository struct {
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	DefaultBranch string     `json:"default_branch"`
	HTMLURL       string     `json:"html_url"`
	Private       bool       `json:"private"`
	Archived      bool       `json:"archived"`
	Owner         *forgeUser `json:"owner,omitempty"`
}

type forgeUser struct {
	Login    string `json:"login,omitempty"`
	UserName string `json:"username,omitempty"`
}

type Label struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Milestone struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type CreateIssueOption struct {
	Assignee  string   `json:"assignee,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Body      string   `json:"body,omitempty"`
	Closed    bool     `json:"closed,omitempty"`
	Deadline  string   `json:"due_date,omitempty"`
	Labels    any      `json:"labels,omitempty"`
	Milestone int64    `json:"milestone,omitempty"`
	Ref       string   `json:"ref,omitempty"`
	Title     string   `json:"title"`
}

type Comment struct {
	ID   int64      `json:"id"`
	Body string     `json:"body,omitempty"`
	User *forgeUser `json:"user,omitempty"`
}

type CreatePullRequestOption struct {
	Assignee  string   `json:"assignee,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Base      string   `json:"base,omitempty"`
	Body      string   `json:"body,omitempty"`
	Head      string   `json:"head,omitempty"`
	Labels    any      `json:"labels,omitempty"`
	Milestone int64    `json:"milestone,omitempty"`
	Title     string   `json:"title,omitempty"`
}

type EditPullRequestOption struct {
	AllowMaintainerEdit bool     `json:"allow_maintainer_edit,omitempty"`
	Assignee            string   `json:"assignee,omitempty"`
	Assignees           []string `json:"assignees,omitempty"`
	Base                string   `json:"base,omitempty"`
	Body                string   `json:"body,omitempty"`
	Labels              any      `json:"labels,omitempty"`
	Milestone           int64    `json:"milestone,omitempty"`
	RemoveDeadline      bool     `json:"unset_due_date,omitempty"`
	State               string   `json:"state,omitempty"`
	Title               string   `json:"title,omitempty"`
}

type WikiPageMetaData struct {
	HTMLURL string `json:"html_url,omitempty"`
	SubURL  string `json:"sub_url,omitempty"`
	Title   string `json:"title,omitempty"`
}

type WikiPage struct {
	ContentBase64 string `json:"content_base64,omitempty"`
	HTMLURL       string `json:"html_url,omitempty"`
	SubURL        string `json:"sub_url,omitempty"`
	Title         string `json:"title,omitempty"`
}

func newForgeClient(url, token string) *forgeClient {
	return &forgeClient{
		baseURL: url,
		token:   token,
	}
}

func (e *forgeAPIError) Error() string {
	if e == nil {
		return "forge API error"
	}
	if e.Message != "" {
		return core.Sprintf("forge %s returned HTTP %d: %s", e.Path, e.StatusCode, e.Message)
	}
	return core.Sprintf("forge %s returned HTTP %d", e.Path, e.StatusCode)
}

func isForgeNotFound(err error) bool {
	var apiErr *forgeAPIError
	return core.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

func (c *forgeClient) getJSON(ctx context.Context, path string, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, out)
}

func (c *forgeClient) postJSON(ctx context.Context, path string, body, out any) error {
	return c.doJSON(ctx, http.MethodPost, path, body, out)
}

func (c *forgeClient) patchJSON(ctx context.Context, path string, body, out any) error {
	return c.doJSON(ctx, http.MethodPatch, path, body, out)
}

func (c *forgeClient) deletePath(ctx context.Context, path string) error {
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *forgeClient) listOrgRepos(ctx context.Context, org string) ([]Repository, error) {
	var repos []Repository
	err := c.getJSON(ctx, core.Sprintf("/api/v1/orgs/%s/repos?limit=50&page=1", org), &repos)
	return repos, err
}

func (c *forgeClient) getRepo(ctx context.Context, org, repo string) (*Repository, error) {
	var item Repository
	if err := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s", org, repo), &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (c *forgeClient) listRepoLabels(ctx context.Context, org, repo string) ([]Label, error) {
	var labels []Label
	err := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/labels?limit=50&page=1", org, repo), &labels)
	return labels, err
}

func (c *forgeClient) mergePullRequest(ctx context.Context, org, repo string, number int64, method string) error {
	if method == "" {
		method = "merge"
	}
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", org, repo, number), map[string]any{
		"Do": method,
	}, nil)
}

func (c *forgeClient) deleteBranch(ctx context.Context, org, repo, branch string) error {
	return c.deletePath(ctx, core.Sprintf("/api/v1/repos/%s/%s/branches/%s", org, repo, core.URLEncode(branch)))
}

func (c *forgeClient) startStopwatch(ctx context.Context, org, repo string, index int64) error {
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues/%d/stopwatch/start", org, repo, index), nil, nil)
}

func (c *forgeClient) stopStopwatch(ctx context.Context, org, repo string, index int64) error {
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues/%d/stopwatch/stop", org, repo, index), nil, nil)
}

func (c *forgeClient) listWikiPages(ctx context.Context, org, repo string) ([]WikiPageMetaData, error) {
	var pages []WikiPageMetaData
	err := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/wiki/pages", org, repo), &pages)
	return pages, err
}

func (c *forgeClient) getWikiPage(ctx context.Context, org, repo, page string) (*WikiPage, error) {
	var wikiPage WikiPage
	err := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/wiki/page/%s", org, repo, core.URLEncode(page)), &wikiPage)
	if err != nil {
		return nil, err
	}
	return &wikiPage, nil
}

func (c *forgeClient) doJSON(ctx context.Context, method, path string, body, out any) error {
	if c == nil {
		return core.E("forgeClient.doJSON", "forge client is required", nil)
	}
	if c.baseURL == "" {
		return core.E("forgeClient.doJSON", "forge base URL is required", nil)
	}

	requestBody := ""
	if body != nil {
		requestBody = core.JSONMarshalString(body)
	}

	var request *http.Request
	var err error
	url := core.Concat(core.TrimSuffix(c.baseURL, "/"), path)
	if requestBody == "" {
		request, err = http.NewRequestWithContext(ctx, method, url, nil)
	} else {
		request, err = http.NewRequestWithContext(ctx, method, url, core.NewReader(requestBody))
	}
	if err != nil {
		return core.E("forgeClient.doJSON", "create request", err)
	}

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", core.Concat("token ", c.token))
	}

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.E("forgeClient.doJSON", "request failed", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		readErr, _ := readResult.Value.(error)
		return core.E("forgeClient.doJSON", "read response", readErr)
	}
	payload, _ := readResult.Value.(string)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return forgeAPIErrorFromResponse(path, response.StatusCode, payload)
	}

	if out == nil || core.Trim(payload) == "" {
		return nil
	}

	parseResult := core.JSONUnmarshalString(payload, out)
	if !parseResult.OK {
		parseErr, _ := parseResult.Value.(error)
		return core.E("forgeClient.doJSON", "parse response", parseErr)
	}
	return nil
}

func forgeAPIErrorFromResponse(path string, statusCode int, payload string) error {
	message := core.Trim(payload)
	if message == "" {
		message = core.Sprintf("HTTP %d", statusCode)
	}

	var envelope struct {
		Message string `json:"message"`
		Error   *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if parseResult := core.JSONUnmarshalString(payload, &envelope); parseResult.OK {
		switch {
		case envelope.Message != "":
			message = envelope.Message
		case envelope.Error != nil && envelope.Error.Message != "":
			message = envelope.Error.Message
		}
	}

	return &forgeAPIError{
		StatusCode: statusCode,
		Path:       path,
		Message:    message,
	}
}
