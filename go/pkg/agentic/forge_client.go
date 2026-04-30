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

type APIError = forgeAPIError

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

func forgeResultError(result core.Result) error { // adapter for external contracts
	if result.OK {
		return nil
	}
	if err, ok := result.Value.(error); ok {
		return err
	}
	return core.E("forge.result", "request failed", nil)
}

func (c *forgeClient) getJSON(ctx context.Context, path string, out any) core.Result {
	return c.doJSON(ctx, http.MethodGet, path, nil, out)
}

func (c *forgeClient) postJSON(ctx context.Context, path string, body, out any) core.Result {
	return c.doJSON(ctx, http.MethodPost, path, body, out)
}

func (c *forgeClient) patchJSON(ctx context.Context, path string, body, out any) core.Result {
	return c.doJSON(ctx, http.MethodPatch, path, body, out)
}

func (c *forgeClient) deletePath(ctx context.Context, path string) core.Result {
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

func (c *forgeClient) listOrgRepos(ctx context.Context, org string) core.Result {
	var repos []Repository
	result := c.getJSON(ctx, core.Sprintf("/api/v1/orgs/%s/repos?limit=50&page=1", org), &repos)
	if !result.OK {
		return result
	}
	return core.Ok(repos)
}

func (c *forgeClient) getRepo(ctx context.Context, org, repo string) core.Result {
	var item Repository
	result := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s", org, repo), &item)
	if !result.OK {
		return result
	}
	return core.Ok(&item)
}

func (c *forgeClient) listRepoLabels(ctx context.Context, org, repo string) core.Result {
	var labels []Label
	result := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/labels?limit=50&page=1", org, repo), &labels)
	if !result.OK {
		return result
	}
	return core.Ok(labels)
}

func (c *forgeClient) mergePullRequest(ctx context.Context, org, repo string, number int64, method string) core.Result {
	if method == "" {
		method = "merge"
	}
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/pulls/%d/merge", org, repo, number), map[string]any{
		"Do": method,
	}, nil)
}

func (c *forgeClient) deleteBranch(ctx context.Context, org, repo, branch string) core.Result {
	return c.deletePath(ctx, core.Sprintf("/api/v1/repos/%s/%s/branches/%s", org, repo, core.URLEncode(branch)))
}

func (c *forgeClient) startStopwatch(ctx context.Context, org, repo string, index int64) core.Result {
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues/%d/stopwatch/start", org, repo, index), nil, nil)
}

func (c *forgeClient) stopStopwatch(ctx context.Context, org, repo string, index int64) core.Result {
	return c.postJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/issues/%d/stopwatch/stop", org, repo, index), nil, nil)
}

func (c *forgeClient) listWikiPages(ctx context.Context, org, repo string) core.Result {
	var pages []WikiPageMetaData
	result := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/wiki/pages", org, repo), &pages)
	if !result.OK {
		return result
	}
	return core.Ok(pages)
}

func (c *forgeClient) getWikiPage(ctx context.Context, org, repo, page string) core.Result {
	var wikiPage WikiPage
	result := c.getJSON(ctx, core.Sprintf("/api/v1/repos/%s/%s/wiki/page/%s", org, repo, core.URLEncode(page)), &wikiPage)
	if !result.OK {
		return result
	}
	return core.Ok(&wikiPage)
}

func (c *forgeClient) doJSON(ctx context.Context, method, path string, body, out any) core.Result {
	if c == nil {
		return core.Fail(core.E("forgeClient.doJSON", "forge client is required", nil))
	}
	if c.baseURL == "" {
		return core.Fail(core.E("forgeClient.doJSON", "forge base URL is required", nil))
	}

	requestBody := ""
	if body != nil {
		requestBody = core.JSONMarshalString(body)
	}

	url := core.Concat(core.TrimSuffix(c.baseURL, "/"), path)
	var requestResult core.Result
	if requestBody == "" {
		requestResult = core.NewHTTPRequestContext(ctx, method, url, nil)
	} else {
		requestResult = core.NewHTTPRequestContext(ctx, method, url, core.NewReader(requestBody))
	}
	if !requestResult.OK {
		err, _ := requestResult.Value.(error)
		return core.Fail(core.E("forgeClient.doJSON", "create request", err))
	}
	request := requestResult.Value.(*core.Request)

	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		request.Header.Set("Authorization", core.Concat("token ", c.token))
	}

	response, err := defaultClient.Do(request)
	if err != nil {
		return core.Fail(core.E("forgeClient.doJSON", "request failed", err))
	}

	readResult := core.ReadAll(response.Body)
	if !readResult.OK {
		readErr, _ := readResult.Value.(error)
		return core.Fail(core.E("forgeClient.doJSON", "read response", readErr))
	}
	payload := readResult.Value.(string)

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return forgeAPIErrorFromResponse(path, response.StatusCode, payload)
	}

	if out == nil || core.Trim(payload) == "" {
		return core.Ok(nil)
	}

	parseResult := core.JSONUnmarshalString(payload, out)
	if !parseResult.OK {
		parseErr, _ := parseResult.Value.(error)
		return core.Fail(core.E("forgeClient.doJSON", "parse response", parseErr))
	}
	return core.Ok(nil)
}

func forgeAPIErrorFromResponse(path string, statusCode int, payload string) core.Result {
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

	return core.Fail(&forgeAPIError{
		StatusCode: statusCode,
		Path:       path,
		Message:    message,
	})
}
