// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"regexp"

	core "dappco.re/go"
)

var pipelineEpicBranchPattern = regexp.MustCompile("Epic branch:\\s*`([^`]+)`")
var pipelineEpicChildPattern = regexp.MustCompile(`(?m)^(\s*-\s*\[)([ xX])(\]\s*#)(\d+)(.*)$`)

type MetaReader struct {
	GetPRMeta           func(ctx context.Context, repo string, prNumber int) (PipelinePRMeta, error)
	GetEpicMeta         func(ctx context.Context, repo string, issueNumber int) (PipelineEpicMeta, error)
	GetIssueState       func(ctx context.Context, repo string, issueNumber int) (PipelineIssueState, error)
	GetCommentReactions func(ctx context.Context, repo string, commentID int64) ([]PipelineReactionMeta, error)
	// ClassifyIssue derives epic / audit / parent signals from the structural
	// fields of an issue record (labels + native sub-issue links + pull_request)
	// rather than regexping the markdown body. This mirrors the PHP
	// ForgejoMetaReader structural-read approach so the Go audit path stays in
	// parity with PHP. Consumers hold a decoded record already (the issue list
	// scan), so the classifier takes the record directly and performs no I/O.
	//
	//	signal := reader.ClassifyIssue(issue)
	//	if signal.IsEpic { ... }
	ClassifyIssue func(issue pipelineIssueRecord) PipelineIssueSignal
}

// PipelineIssueSignal is the structural classification of an issue. Every field
// is derived from typed API fields (labels, sub-issue links, pull_request),
// never from parsing the body prose. ParentNumber is 0 when the issue has no
// structurally-linked parent epic.
type PipelineIssueSignal struct {
	Number       int      `json:"number"`
	IsAudit      bool     `json:"is_audit"`
	IsEpic       bool     `json:"is_epic"`
	IsPR         bool     `json:"is_pr"`
	HasParent    bool     `json:"has_parent"`
	ParentNumber int      `json:"parent_number,omitempty"`
	Labels       []string `json:"labels,omitempty"`
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

func (s *PrepSubsystem) cmdPipelineMonitor(options core.Options) core.Result {
	ctx := s.commandContext()
	output, err := pipelineMonitorWithReader(s, ctx, PipelineMonitorInput{
		Org:    pipelineOrgValue(options),
		Repo:   pipelineRepoValue(options),
		DryRun: optionBoolValue(options, "dry_run", "dry-run"),
	}, newPipelineForgeMetaReader(s, pipelineOrgValue(options)))
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

var pipelineMonitorWithReader = func(s *PrepSubsystem, ctx context.Context, input PipelineMonitorInput, reader *MetaReader) (PipelineMonitorOutput, error) {
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
		records, err := pipelineListOrgRepos(s, ctx, input.Org)
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
		pullRequests, err := pipelineListPullRequests(s, ctx, input.Org, repo, "open")
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

var pipelineListPullRequests = func(s *PrepSubsystem, ctx context.Context, org, repo, state string) ([]pipelinePullRequestRecord, error) {
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

var newPipelineForgeMetaReader = func(s *PrepSubsystem, org string) *MetaReader {
	reader := &MetaReader{}
	reader.ClassifyIssue = pipelineClassifyIssueStructural
	reader.GetPRMeta = func(ctx context.Context, repo string, prNumber int) (PipelinePRMeta, error) {
		url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d", s.forgeURL, org, repo, prNumber)
		result := HTTPGet(ctx, url, s.forgeToken, "token")
		if !result.OK {
			return PipelinePRMeta{}, core.E("MetaReader.GetPRMeta", core.Concat("failed to read PR #", core.Sprint(prNumber)), nil)
		}

		var pullRequest pipelinePullRequestRecord
		if err := pipelineDecodeJSON(result.Value.(string), &pullRequest, "MetaReader.GetPRMeta", "failed to decode pull request"); err != nil {
			return PipelinePRMeta{}, err
		}

		statuses := []PipelineCheckMeta{}
		if pullRequest.Head.SHA != "" {
			statusURL := core.Sprintf("%s/api/v1/repos/%s/%s/commits/%s/status", s.forgeURL, org, repo, pullRequest.Head.SHA)
			statusResult := HTTPGet(ctx, statusURL, s.forgeToken, "token")
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
	reader.GetIssueState = func(ctx context.Context, repo string, issueNumber int) (PipelineIssueState, error) {
		issue, err := pipelineGetIssue(s, ctx, org, repo, issueNumber)
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
	reader.GetEpicMeta = func(ctx context.Context, repo string, issueNumber int) (PipelineEpicMeta, error) {
		issue, err := pipelineGetIssue(s, ctx, org, repo, issueNumber)
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

			state, stateErr := reader.GetIssueState(ctx, repo, number)
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
	reader.GetCommentReactions = func(ctx context.Context, repo string, commentID int64) ([]PipelineReactionMeta, error) {
		url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/comments/%d/reactions", s.forgeURL, org, repo, commentID)
		result := HTTPGet(ctx, url, s.forgeToken, "token")
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
	return reader
}

// pipelineClassifyIssueStructural derives epic / audit / PR / parent signals
// from the typed fields of an issue record. It mirrors PHP's ForgejoMetaReader,
// which classifies from structured API data (labels, native sub-issue links,
// pull_request) and explicitly leaves body prose-parsing out of scope. An issue
// is an epic when it carries the structural `epic` label or has native
// sub-issue children; it is a parent's child when it appears in a sub-issue
// link that names its own parent. No regexp touches the body here.
//
//	signal := pipelineClassifyIssueStructural(issue)
//	if signal.IsEpic { ... }
func pipelineClassifyIssueStructural(issue pipelineIssueRecord) PipelineIssueSignal {
	labels := pipelineIssueLabelNames(issue)
	children := pipelineIssueStructuralChildren(issue)

	signal := PipelineIssueSignal{
		Number:  issue.Number,
		IsAudit: pipelineLabelsContain(labels, "audit"),
		IsEpic:  pipelineLabelsContain(labels, "epic") || len(children) > 0,
		IsPR:    len(issue.PullRequest) > 0,
		Labels:  labels,
	}
	return signal
}

// pipelineIssueStructuralChildren returns the structurally-linked child issue
// numbers of an epic, reading the native sub-issue arrays (subtasks first, then
// sub_issues) the same way PHP ForgejoMetaReader::extractEpicChildren does.
// Absence of both arrays yields an empty slice — it is not an error.
func pipelineIssueStructuralChildren(issue pipelineIssueRecord) []int {
	records := issue.SubTasks
	if len(records) == 0 {
		records = issue.SubIssues
	}

	numbers := make([]int, 0, len(records))
	for _, record := range records {
		number := record.IssueID
		if number == 0 {
			number = record.Number
		}
		if number > 0 {
			numbers = append(numbers, number)
		}
	}
	return numbers
}

func pipelineLabelsContain(labels []string, want string) bool {
	for _, name := range labels {
		if core.Lower(name) == core.Lower(want) {
			return true
		}
	}
	return false
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
