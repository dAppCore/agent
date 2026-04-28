// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ScanInput struct {
	Org    string   `json:"org,omitempty"`
	Labels []string `json:"labels,omitempty"`
	Limit  int      `json:"limit,omitempty"`
}

type ScanOutput struct {
	Success bool        `json:"success"`
	Count   int         `json:"count"`
	Issues  []ScanIssue `json:"issues"`
}

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

	repos, err := s.listOrgRepos(ctx, input.Org)
	if err != nil {
		return nil, ScanOutput{}, err
	}

	for _, repoName := range repos {
		for _, label := range input.Labels {
			issues, err := s.listRepoIssues(ctx, input.Org, repoName, label)
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
	repos, err := s.forge.Repos.ListOrgRepos(ctx, org)
	if err != nil {
		return nil, core.E("scan.listOrgRepos", "failed to list repos", err)
	}

	var allNames []string
	for _, repoInfo := range repos {
		allNames = append(allNames, repoInfo.Name)
	}
	return allNames, nil
}

func (s *PrepSubsystem) listRepoIssues(ctx context.Context, org, repo, label string) ([]ScanIssue, error) {
	requestURL := core.Sprintf("%s/api/v1/repos/%s/%s/issues?state=open&limit=10&type=issues",
		s.forgeURL, org, repo)
	if label != "" {
		requestURL = core.Concat(requestURL, "&labels=", core.URLEncode(label))
	}
	httpResult := HTTPGet(ctx, requestURL, s.forgeToken, "token")
	if !httpResult.OK {
		return nil, core.E("scan.listRepoIssues", core.Concat("failed to list issues for ", repo), nil)
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
	if parseResult := core.JSONUnmarshalString(httpResult.Value.(string), &issues); !parseResult.OK {
		err, _ := parseResult.Value.(error)
		return nil, core.E("scan.listRepoIssues", "parse issues response", err)
	}

	var result []ScanIssue
	for _, issue := range issues {
		var labels []string
		for _, labelInfo := range issue.Labels {
			labels = append(labels, labelInfo.Name)
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
