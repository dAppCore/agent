// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"net/http"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ScanInput is the input for agentic_scan.
//
//	input := agentic.ScanInput{Org: "core", Labels: []string{"agentic", "bug"}, Limit: 20}
type ScanInput struct {
	Org    string   `json:"org,omitempty"`    // default "core"
	Labels []string `json:"labels,omitempty"` // filter by labels (default: agentic, help-wanted, bug)
	Limit  int      `json:"limit,omitempty"`  // max issues to return
}

// ScanOutput is the output for agentic_scan.
//
//	out := agentic.ScanOutput{Success: true, Count: 1, Issues: []agentic.ScanIssue{{Repo: "go-io", Number: 12}}}
type ScanOutput struct {
	Success bool        `json:"success"`
	Count   int         `json:"count"`
	Issues  []ScanIssue `json:"issues"`
}

// ScanIssue is a single actionable issue.
//
//	issue := agentic.ScanIssue{Repo: "go-io", Number: 12, Title: "Replace fmt.Errorf"}
type ScanIssue struct {
	Repo     string   `json:"repo"`
	Number   int      `json:"number"`
	Title    string   `json:"title"`
	Labels   []string `json:"labels"`
	Assignee string   `json:"assignee,omitempty"`
	URL      string   `json:"url"`
}

func (s *PrepSubsystem) scan(ctx context.Context, _ *mcp.CallToolRequest, input ScanInput) (*mcp.CallToolResult, ScanOutput, error) {
	if s.forgeToken == "" {
		return nil, ScanOutput{}, core.E("scan", "no Forge token configured", nil)
	}

	if input.Org == "" {
		input.Org = "core"
	}
	if input.Limit == 0 {
		input.Limit = 20
	}
	if len(input.Labels) == 0 {
		input.Labels = []string{"agentic", "help-wanted", "bug"}
	}

	var allIssues []ScanIssue

	// Get repos for the org
	repos, err := s.listOrgRepos(ctx, input.Org)
	if err != nil {
		return nil, ScanOutput{}, err
	}

	for _, repo := range repos {
		for _, label := range input.Labels {
			issues, err := s.listRepoIssues(ctx, input.Org, repo, label)
			if err != nil {
				continue
			}
			allIssues = append(allIssues, issues...)

			if len(allIssues) >= input.Limit {
				break
			}
		}
		if len(allIssues) >= input.Limit {
			break
		}
	}

	// Deduplicate by repo+number
	seen := make(map[string]bool)
	var unique []ScanIssue
	for _, issue := range allIssues {
		key := core.Sprintf("%s#%d", issue.Repo, issue.Number)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, issue)
		}
	}

	if len(unique) > input.Limit {
		unique = unique[:input.Limit]
	}

	return nil, ScanOutput{
		Success: true,
		Count:   len(unique),
		Issues:  unique,
	}, nil
}

func (s *PrepSubsystem) listOrgRepos(ctx context.Context, org string) ([]string, error) {
	var allNames []string
	page := 1

	for {
		u := core.Sprintf("%s/api/v1/orgs/%s/repos?limit=50&page=%d", s.forgeURL, org, page)
		req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
		if err != nil {
			return nil, core.E("scan.listOrgRepos", "failed to create request", err)
		}
		req.Header.Set("Authorization", "token "+s.forgeToken)

		resp, err := s.client.Do(req)
		if err != nil {
			return nil, core.E("scan.listOrgRepos", "failed to list repos", err)
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, core.E("scan.listOrgRepos", core.Sprintf("HTTP %d listing repos", resp.StatusCode), nil)
		}

		var repos []struct {
			Name string `json:"name"`
		}
		json.NewDecoder(resp.Body).Decode(&repos)
		resp.Body.Close()

		for _, r := range repos {
			allNames = append(allNames, r.Name)
		}

		// If we got fewer than the limit, we've reached the last page
		if len(repos) < 50 {
			break
		}
		page++
	}
	return allNames, nil
}

func (s *PrepSubsystem) listRepoIssues(ctx context.Context, org, repo, label string) ([]ScanIssue, error) {
	u := core.Sprintf("%s/api/v1/repos/%s/%s/issues?state=open&limit=10&type=issues",
		s.forgeURL, org, repo)
	if label != "" {
		u += "&labels=" + core.Replace(core.Replace(label, " ", "%20"), "&", "%26")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, core.E("scan.listRepoIssues", "failed to create request", err)
	}
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, core.E("scan.listRepoIssues", "failed to list issues for "+repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, core.E("scan.listRepoIssues", core.Sprintf("HTTP %d listing issues for %s", resp.StatusCode, repo), nil)
	}

	var issues []struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignee *struct {
			Login string `json:"login"`
		} `json:"assignee"`
		HTMLURL string `json:"html_url"`
	}
	json.NewDecoder(resp.Body).Decode(&issues)

	var result []ScanIssue
	for _, issue := range issues {
		var labels []string
		for _, l := range issue.Labels {
			labels = append(labels, l.Name)
		}
		assignee := ""
		if issue.Assignee != nil {
			assignee = issue.Assignee.Login
		}

		result = append(result, ScanIssue{
			Repo:     repo,
			Number:   issue.Number,
			Title:    issue.Title,
			Labels:   labels,
			Assignee: assignee,
			URL:      core.Replace(issue.HTMLURL, "https://forge.lthn.ai", s.forgeURL),
		})
	}

	return result, nil
}
