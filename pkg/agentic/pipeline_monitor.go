// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"regexp"

	core "dappco.re/go/core"
)

var pipelineEpicBranchPattern = regexp.MustCompile("Epic branch:\\s*`([^`]+)`")
var pipelineEpicChildPattern = regexp.MustCompile(`(?m)^(\s*-\s*\[)([ xX])(\]\s*#)(\d+)(.*)$`)

type MetaReader interface {
	GetPRMeta(ctx context.Context, repo string, prNumber int) (PipelinePRMeta, error)
	GetEpicMeta(ctx context.Context, repo string, issueNumber int) (PipelineEpicMeta, error)
	GetIssueState(ctx context.Context, repo string, issueNumber int) (PipelineIssueState, error)
	GetCommentReactions(ctx context.Context, repo string, commentID int64) ([]PipelineReactionMeta, error)
}

type PipelineCheckMeta struct {
	Name       string `json:"name"`
	Conclusion string `json:"conclusion,omitempty"`
	Status     string `json:"status,omitempty"`
}

type PipelinePRMeta struct {
	Number          int                 `json:"number"`
	State           string              `json:"state"`
	Mergeable       string              `json:"mergeable"`
	HeadSHA         string              `json:"head_sha,omitempty"`
	HeadDate        string              `json:"head_date,omitempty"`
	BaseBranch      string              `json:"base_branch,omitempty"`
	HeadBranch      string              `json:"head_branch,omitempty"`
	Checks          []PipelineCheckMeta `json:"checks,omitempty"`
	ThreadsTotal    int                 `json:"threads_total,omitempty"`
	ThreadsResolved int                 `json:"threads_resolved,omitempty"`
	HasEyesReaction bool                `json:"has_eyes_reaction,omitempty"`
	URL             string              `json:"url,omitempty"`
}

type PipelineIssueState struct {
	Number int      `json:"number"`
	State  string   `json:"state"`
	Title  string   `json:"title"`
	URL    string   `json:"url,omitempty"`
	Labels []string `json:"labels,omitempty"`
}

type PipelineEpicChildMeta struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Checked bool   `json:"checked"`
	State   string `json:"state"`
	URL     string `json:"url,omitempty"`
	PRs     []int  `json:"prs,omitempty"`
}

type PipelineEpicMeta struct {
	Number   int                     `json:"number"`
	Title    string                  `json:"title"`
	State    string                  `json:"state"`
	URL      string                  `json:"url,omitempty"`
	Body     string                  `json:"body,omitempty"`
	Branch   string                  `json:"branch,omitempty"`
	Children []PipelineEpicChildMeta `json:"children"`
}

type PipelineReactionMeta struct {
	Content string `json:"content"`
	Count   int    `json:"count"`
}

type PipelineMonitorInput struct {
	Org    string `json:"org,omitempty"`
	Repo   string `json:"repo,omitempty"`
	DryRun bool   `json:"dry_run,omitempty"`
}

type PipelineMonitorAction struct {
	Repo   string `json:"repo"`
	Number int    `json:"number"`
	Action string `json:"action"`
	Reason string `json:"reason"`
	URL    string `json:"url,omitempty"`
}

type PipelineMonitorOutput struct {
	Success bool                    `json:"success"`
	Org     string                  `json:"org,omitempty"`
	Repo    string                  `json:"repo,omitempty"`
	Actions []PipelineMonitorAction `json:"actions,omitempty"`
}

type pipelinePullRequestRecord struct {
	Number                  int            `json:"number"`
	Title                   string         `json:"title"`
	State                   string         `json:"state"`
	HTMLURL                 string         `json:"html_url"`
	Merged                  bool           `json:"merged"`
	Mergeable               *bool          `json:"mergeable"`
	MergeableState          string         `json:"mergeable_state"`
	UpdatedAt               string         `json:"updated_at"`
	ReviewThreadsTotal      int            `json:"review_threads_total"`
	ReviewThreadsResolved   int            `json:"review_threads_resolved"`
	ReviewThreadsUnresolved int            `json:"review_threads_unresolved"`
	ReviewComments          int            `json:"review_comments"`
	Comments                int            `json:"comments"`
	Reactions               map[string]int `json:"reactions"`
	Head                    struct {
		Ref       string `json:"ref"`
		SHA       string `json:"sha"`
		Date      string `json:"date"`
		UpdatedAt string `json:"updated_at"`
		Repo      struct {
			UpdatedAt string `json:"updated_at"`
			PushedAt  string `json:"pushed_at"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
}

type pipelineStatusRecord struct {
	Statuses []struct {
		Context    string `json:"context"`
		Name       string `json:"name"`
		Status     string `json:"status"`
		State      string `json:"state"`
		Conclusion string `json:"conclusion"`
		UpdatedAt  string `json:"updated_at"`
		CreatedAt  string `json:"created_at"`
	} `json:"statuses"`
}

type pipelineForgeMetaReader struct {
	subsystem *PrepSubsystem
	org       string
}

func (s *PrepSubsystem) cmdPipelineMonitor(options core.Options) core.Result {
	ctx := s.commandContext()
	output, err := s.pipelineMonitorWithReader(ctx, PipelineMonitorInput{
		Org:    pipelineOrgValue(options),
		Repo:   pipelineRepoValue(options),
		DryRun: optionBoolValue(options, "dry_run", "dry-run"),
	}, &pipelineForgeMetaReader{subsystem: s, org: pipelineOrgValue(options)})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.Repo != "" {
		core.Print(nil, "repo:    %s/%s", output.Org, output.Repo)
	} else {
		core.Print(nil, "org:     %s", output.Org)
	}
	core.Print(nil, "actions: %d", len(output.Actions))
	for _, action := range output.Actions {
		core.Print(nil, "  %s #%d %s", action.Repo, action.Number, action.Action)
	}
	if len(output.Actions) == 0 {
		core.Print(nil, "no interventions")
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) pipelineMonitorWithReader(ctx context.Context, input PipelineMonitorInput, reader MetaReader) (PipelineMonitorOutput, error) {
	if s.forgeToken == "" {
		return PipelineMonitorOutput{}, core.E("pipelineMonitor", "no Forge token configured", nil)
	}
	if input.Org == "" {
		input.Org = "core"
	}

	repos := []string{}
	if input.Repo != "" {
		repos = append(repos, input.Repo)
	} else {
		records, err := s.pipelineListOrgRepos(ctx, input.Org)
		if err != nil {
			return PipelineMonitorOutput{}, err
		}
		for _, record := range records {
			repos = append(repos, record.Name)
		}
	}

	output := PipelineMonitorOutput{
		Success: true,
		Org:     input.Org,
		Repo:    input.Repo,
	}

	for _, repo := range repos {
		pullRequests, err := s.pipelineListPullRequests(ctx, input.Org, repo, "open")
		if err != nil {
			return PipelineMonitorOutput{}, err
		}

		for _, pullRequest := range pullRequests {
			meta, err := reader.GetPRMeta(ctx, repo, pullRequest.Number)
			if err != nil {
				return PipelineMonitorOutput{}, err
			}
			if meta.State != "open" {
				continue
			}

			action := PipelineMonitorAction{
				Repo:   repo,
				Number: meta.Number,
				URL:    meta.URL,
			}

			switch {
			case meta.Mergeable == "conflicting":
				action.Action = "fix/conflicts"
				action.Reason = "merge conflict detected"
				if !input.DryRun {
					s.commentOnIssue(ctx, input.Org, repo, meta.Number, "Can you fix the merge conflict?")
				}
			case meta.ThreadsTotal > meta.ThreadsResolved:
				action.Action = "fix/reviews"
				action.Reason = "unresolved review threads detected"
				if !input.DryRun {
					s.commentOnIssue(ctx, input.Org, repo, meta.Number, "Can you fix the code reviews?")
				}
			case pipelineChecksSuccessful(meta.Checks):
				action.Action = "merge"
				action.Reason = "all checks passed and no review blockers remain"
				if !input.DryRun {
					mergeResult := s.forgeMergePR(ctx, input.Org, repo, meta.Number)
					if !mergeResult.OK {
						return PipelineMonitorOutput{}, commandResultError("pipelineMonitor.merge", mergeResult)
					}
				}
			default:
				continue
			}

			output.Actions = append(output.Actions, action)
		}
	}

	return output, nil
}

func (s *PrepSubsystem) pipelineListPullRequests(ctx context.Context, org, repo, state string) ([]pipelinePullRequestRecord, error) {
	if state == "" {
		state = "open"
	}

	url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=%s&limit=100&page=1", s.forgeURL, org, repo, state)
	result := HTTPGet(ctx, url, s.forgeToken, "token")
	if !result.OK {
		return nil, core.E("pipelineListPullRequests", core.Concat("failed to list pull requests for ", repo), nil)
	}

	var pullRequests []pipelinePullRequestRecord
	if err := pipelineDecodeJSON(result.Value.(string), &pullRequests, "pipelineListPullRequests", "failed to decode pull request list"); err != nil {
		return nil, err
	}
	return pullRequests, nil
}

func (r *pipelineForgeMetaReader) GetPRMeta(ctx context.Context, repo string, prNumber int) (PipelinePRMeta, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d", r.subsystem.forgeURL, r.org, repo, prNumber)
	result := HTTPGet(ctx, url, r.subsystem.forgeToken, "token")
	if !result.OK {
		return PipelinePRMeta{}, core.E("MetaReader.GetPRMeta", core.Concat("failed to read PR #", core.Sprint(prNumber)), nil)
	}

	var pullRequest pipelinePullRequestRecord
	if err := pipelineDecodeJSON(result.Value.(string), &pullRequest, "MetaReader.GetPRMeta", "failed to decode pull request"); err != nil {
		return PipelinePRMeta{}, err
	}

	statuses := []PipelineCheckMeta{}
	if pullRequest.Head.SHA != "" {
		statusURL := core.Sprintf("%s/api/v1/repos/%s/%s/commits/%s/status", r.subsystem.forgeURL, r.org, repo, pullRequest.Head.SHA)
		statusResult := HTTPGet(ctx, statusURL, r.subsystem.forgeToken, "token")
		if statusResult.OK {
			var status pipelineStatusRecord
			if err := pipelineDecodeJSON(statusResult.Value.(string), &status, "MetaReader.GetPRMeta", "failed to decode commit status"); err == nil {
				for _, entry := range status.Statuses {
					rawState := entry.Status
					if rawState == "" {
						rawState = entry.State
					}
					if rawState == "" {
						rawState = entry.Conclusion
					}

					name := entry.Context
					if name == "" {
						name = entry.Name
					}
					statuses = append(statuses, PipelineCheckMeta{
						Name:       name,
						Conclusion: pipelineCheckConclusion(rawState),
						Status:     pipelineCheckStatus(rawState),
					})
				}
			}
		}
	}

	state := core.Lower(pullRequest.State)
	if pullRequest.Merged {
		state = "merged"
	}
	if state == "" {
		state = "open"
	}

	mergeable := "unknown"
	switch {
	case pullRequest.Mergeable != nil && *pullRequest.Mergeable:
		mergeable = "mergeable"
	case pullRequest.Mergeable != nil && !*pullRequest.Mergeable:
		mergeable = "conflicting"
	case core.Lower(pullRequest.MergeableState) == "clean", core.Lower(pullRequest.MergeableState) == "mergeable":
		mergeable = "mergeable"
	case core.Lower(pullRequest.MergeableState) == "dirty", core.Lower(pullRequest.MergeableState) == "conflicting":
		mergeable = "conflicting"
	}

	threadsTotal := pullRequest.ReviewThreadsTotal
	if threadsTotal == 0 {
		if pullRequest.ReviewComments > 0 {
			threadsTotal = pullRequest.ReviewComments
		} else {
			threadsTotal = pullRequest.Comments
		}
	}

	threadsResolved := pullRequest.ReviewThreadsResolved
	if threadsResolved == 0 && pullRequest.ReviewThreadsUnresolved > 0 && threadsTotal >= pullRequest.ReviewThreadsUnresolved {
		threadsResolved = threadsTotal - pullRequest.ReviewThreadsUnresolved
	}

	headDate := pullRequest.Head.Date
	if headDate == "" {
		headDate = pullRequest.Head.UpdatedAt
	}
	if headDate == "" {
		headDate = pullRequest.Head.Repo.UpdatedAt
	}
	if headDate == "" {
		headDate = pullRequest.Head.Repo.PushedAt
	}
	if headDate == "" {
		headDate = pullRequest.UpdatedAt
	}

	return PipelinePRMeta{
		Number:          pullRequest.Number,
		State:           state,
		Mergeable:       mergeable,
		HeadSHA:         pullRequest.Head.SHA,
		HeadDate:        headDate,
		BaseBranch:      pullRequest.Base.Ref,
		HeadBranch:      pullRequest.Head.Ref,
		Checks:          statuses,
		ThreadsTotal:    threadsTotal,
		ThreadsResolved: threadsResolved,
		HasEyesReaction: pullRequest.Reactions["eyes"] > 0,
		URL:             pullRequest.HTMLURL,
	}, nil
}

func (r *pipelineForgeMetaReader) GetEpicMeta(ctx context.Context, repo string, issueNumber int) (PipelineEpicMeta, error) {
	issue, err := r.subsystem.pipelineGetIssue(ctx, r.org, repo, issueNumber)
	if err != nil {
		return PipelineEpicMeta{}, err
	}

	meta := PipelineEpicMeta{
		Number: issue.Number,
		Title:  issue.Title,
		State:  issue.State,
		URL:    issue.HTMLURL,
		Body:   issue.Body,
	}

	branchMatch := pipelineEpicBranchPattern.FindStringSubmatch(issue.Body)
	if len(branchMatch) == 2 {
		meta.Branch = branchMatch[1]
	}

	matches := pipelineEpicChildPattern.FindAllStringSubmatch(issue.Body, -1)
	for _, match := range matches {
		if len(match) != 6 {
			continue
		}
		number := parseIntString(match[4])
		if number <= 0 {
			continue
		}

		state, stateErr := r.GetIssueState(ctx, repo, number)
		if stateErr != nil {
			return PipelineEpicMeta{}, stateErr
		}
		meta.Children = append(meta.Children, PipelineEpicChildMeta{
			Number:  number,
			Title:   state.Title,
			Checked: core.Lower(match[2]) == "x",
			State:   state.State,
			URL:     state.URL,
		})
	}

	return meta, nil
}

func (r *pipelineForgeMetaReader) GetIssueState(ctx context.Context, repo string, issueNumber int) (PipelineIssueState, error) {
	issue, err := r.subsystem.pipelineGetIssue(ctx, r.org, repo, issueNumber)
	if err != nil {
		return PipelineIssueState{}, err
	}

	state := issue.State
	if state == "" {
		state = "open"
	}

	return PipelineIssueState{
		Number: issue.Number,
		State:  state,
		Title:  issue.Title,
		URL:    issue.HTMLURL,
		Labels: pipelineIssueLabelNames(issue),
	}, nil
}

func (r *pipelineForgeMetaReader) GetCommentReactions(ctx context.Context, repo string, commentID int64) ([]PipelineReactionMeta, error) {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/comments/%d/reactions", r.subsystem.forgeURL, r.org, repo, commentID)
	result := HTTPGet(ctx, url, r.subsystem.forgeToken, "token")
	if !result.OK {
		return nil, core.E("MetaReader.GetCommentReactions", core.Concat("failed to read comment reactions for ", core.Sprint(commentID)), nil)
	}

	var reactions []struct {
		Content string `json:"content"`
	}
	if err := pipelineDecodeJSON(result.Value.(string), &reactions, "MetaReader.GetCommentReactions", "failed to decode reactions"); err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, reaction := range reactions {
		counts[reaction.Content]++
	}

	aggregated := []PipelineReactionMeta{}
	for content, count := range counts {
		aggregated = append(aggregated, PipelineReactionMeta{Content: content, Count: count})
	}
	return aggregated, nil
}

func pipelineCheckConclusion(rawState string) string {
	switch core.Lower(rawState) {
	case "success":
		return "success"
	case "failure", "error":
		return "failure"
	default:
		return ""
	}
}

func pipelineCheckStatus(rawState string) string {
	switch core.Lower(rawState) {
	case "success", "failure", "error":
		return "completed"
	case "pending", "queued":
		return "queued"
	case "running", "in_progress":
		return "in_progress"
	default:
		return ""
	}
}

func pipelineChecksSuccessful(checks []PipelineCheckMeta) bool {
	if len(checks) == 0 {
		return false
	}
	for _, check := range checks {
		if check.Status != "completed" || check.Conclusion != "success" {
			return false
		}
	}
	return true
}
