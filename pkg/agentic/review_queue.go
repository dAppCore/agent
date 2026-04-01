// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"regexp"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// input := agentic.ReviewQueueInput{Reviewer: "coderabbit", Limit: 4, DryRun: true}
type ReviewQueueInput struct {
	Limit     int    `json:"limit,omitempty"`
	Reviewer  string `json:"reviewer,omitempty"`
	DryRun    bool   `json:"dry_run,omitempty"`
	LocalOnly bool   `json:"local_only,omitempty"`
}

// out := agentic.ReviewQueueOutput{Success: true, Processed: []agentic.ReviewResult{{Repo: "go-io", Verdict: "clean"}}}
type ReviewQueueOutput struct {
	Success   bool           `json:"success"`
	Processed []ReviewResult `json:"processed"`
	Skipped   []string       `json:"skipped,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
}

// result := agentic.ReviewResult{Repo: "go-io", Verdict: "findings", Findings: 3, Action: "fix_dispatched"}
type ReviewResult struct {
	Repo     string `json:"repo"`
	Verdict  string `json:"verdict"`
	Findings int    `json:"findings"`
	Action   string `json:"action"`
	Detail   string `json:"detail,omitempty"`
}

// limit := agentic.RateLimitInfo{Limited: true, Message: "retry after 2026-03-22T06:00:00Z"}
type RateLimitInfo struct {
	Limited bool      `json:"limited"`
	RetryAt time.Time `json:"retry_at,omitempty"`
	Message string    `json:"message,omitempty"`
}

var retryAfterPattern = compileRetryAfterPattern()

func compileRetryAfterPattern() *regexp.Regexp {
	pattern, err := regexp.Compile(`(\d+)\s*minutes?\s*(?:and\s*)?(\d+)?\s*seconds?`)
	if err != nil {
		return nil
	}
	return pattern
}

func (s *PrepSubsystem) registerReviewQueueTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_review_queue",
		Description: "Process the review queue. Supports coderabbit, codex, or both reviewers, auto-merges clean ones on GitHub, dispatches fix agents for findings, and respects rate limits.",
	}, s.reviewQueue)
}

// reviewers := reviewQueueReviewers("both")
func reviewQueueReviewers(reviewer string) []string {
	switch core.Lower(core.Trim(reviewer)) {
	case "codex":
		return []string{"codex"}
	case "both":
		return []string{"codex", "coderabbit"}
	default:
		return []string{"coderabbit"}
	}
}

// result := c.Command("pr-manage").Run(ctx, core.NewOptions(
//
//	core.Option{Key: "limit", Value: 4},
//
// ))
func (s *PrepSubsystem) cmdPRManage(options core.Options) core.Result {
	ctx := s.commandContext()
	input := ReviewQueueInput{
		Limit:     optionIntValue(options, "limit"),
		Reviewer:  optionStringValue(options, "reviewer"),
		DryRun:    optionBoolValue(options, "dry-run"),
		LocalOnly: optionBoolValue(options, "local-only"),
	}

	_, output, err := s.reviewQueue(ctx, nil, input)
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	if output.RateLimit != nil && output.RateLimit.Message != "" {
		core.Print(nil, "rate limit: %s", output.RateLimit.Message)
	}
	for _, item := range output.Processed {
		core.Print(nil, "%s: %s (%s)", item.Repo, item.Verdict, item.Action)
	}
	for _, item := range output.Skipped {
		core.Print(nil, "skipped: %s", item)
	}

	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) reviewQueue(ctx context.Context, _ *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 4
	}

	basePath := s.codePath
	if basePath == "" {
		basePath = core.JoinPath(HomeDir(), "Code")
	}
	basePath = core.JoinPath(basePath, "core")

	candidates := s.findReviewCandidates(basePath)
	if len(candidates) == 0 {
		return nil, ReviewQueueOutput{
			Success:   true,
			Processed: nil,
		}, nil
	}

	var processed []ReviewResult
	var skipped []string
	var rateInfo *RateLimitInfo

	for _, repo := range candidates {
		repoDir := core.JoinPath(basePath, repo)
		for _, reviewer := range reviewQueueReviewers(input.Reviewer) {
			if len(processed) >= limit {
				skipped = append(skipped, core.Concat(repo, " (limit reached)"))
				break
			}

			if reviewer == "coderabbit" && rateInfo != nil && rateInfo.Limited && time.Now().Before(rateInfo.RetryAt) {
				skipped = append(skipped, core.Concat(repo, " (rate limited)"))
				continue
			}

			result := s.reviewRepo(ctx, repoDir, repo, reviewer, input.DryRun, input.LocalOnly)

			if result.Verdict == "rate_limited" {
				if reviewer == "coderabbit" {
					retryAfter := parseRetryAfter(result.Detail)
					rateInfo = &RateLimitInfo{
						Limited: true,
						RetryAt: time.Now().Add(retryAfter),
						Message: result.Detail,
					}
					skipped = append(skipped, core.Concat(repo, " (rate limited: ", retryAfter.String(), ")"))
				}
				continue
			}

			processed = append(processed, result)
		}
	}

	if rateInfo != nil {
		s.saveRateLimitState(rateInfo)
	}

	return nil, ReviewQueueOutput{
		Success:   true,
		Processed: processed,
		Skipped:   skipped,
		RateLimit: rateInfo,
	}, nil
}

// repos := s.findReviewCandidates("/srv/Code/core")
func (s *PrepSubsystem) findReviewCandidates(basePath string) []string {
	paths := core.PathGlob(core.JoinPath(basePath, "*"))

	var candidates []string
	for _, p := range paths {
		if !fs.IsDir(p) {
			continue
		}
		name := core.PathBase(p)
		if !s.hasRemote(p, "github") {
			continue
		}
		ahead := s.commitsAhead(p, "github/main", "HEAD")
		if ahead > 0 {
			candidates = append(candidates, name)
		}
	}
	return candidates
}

// result := s.reviewRepo(ctx, repoDir, "go-io", "coderabbit", false, false)
func (s *PrepSubsystem) reviewRepo(ctx context.Context, repoDir, repo, reviewer string, dryRun, localOnly bool) ReviewResult {
	result := ReviewResult{Repo: repo}
	process := s.Core().Process()

	if reviewer != "codex" {
		if rl := s.loadRateLimitState(); rl != nil && rl.Limited && time.Now().Before(rl.RetryAt) {
			result.Verdict = "rate_limited"
			result.Detail = core.Sprintf("retry after %s", rl.RetryAt.Format(time.RFC3339))
			return result
		}
	}

	if reviewer == "" {
		reviewer = "coderabbit"
	}
	command, args := s.buildReviewCommand(repoDir, reviewer)
	r := process.RunIn(ctx, repoDir, command, args...)
	output, _ := r.Value.(string)

	if core.Contains(output, "Rate limit exceeded") || core.Contains(output, "rate limit") {
		result.Verdict = "rate_limited"
		result.Detail = output
		return result
	}

	if !r.OK && !core.Contains(output, "No findings") && !core.Contains(output, "no issues") {
		result.Verdict = "error"
		result.Detail = output
		return result
	}

	s.storeReviewOutput(repoDir, repo, reviewer, output)

	if core.Contains(output, "No findings") || core.Contains(output, "no issues") || core.Contains(output, "LGTM") {
		result.Verdict = "clean"
		result.Findings = 0

		if dryRun {
			result.Action = "skipped (dry run)"
			return result
		}

		if localOnly {
			result.Action = "clean (local only)"
			return result
		}

		if err := s.pushAndMerge(ctx, repoDir, repo); err != nil {
			result.Action = core.Concat("push failed: ", err.Error())
		} else {
			result.Action = "merged"
		}
	} else {
		result.Verdict = "findings"
		result.Findings = countFindings(output)
		result.Detail = truncate(output, 500)

		if dryRun {
			result.Action = "skipped (dry run)"
			return result
		}

		findingsFile := core.JoinPath(repoDir, ".core", "coderabbit-findings.txt")
		fs.Write(findingsFile, output)

		task := core.Sprintf(
			"Fix CodeRabbit findings. The review output is in .core/coderabbit-findings.txt. Read it, verify each finding against the code, fix what's valid. Run tests. Commit: fix(coderabbit): address review findings\n\nFindings summary (%d issues):\n%s",
			result.Findings, truncate(output, 1500))

		if err := s.dispatchFixFromQueue(ctx, repo, task); err != nil {
			result.Action = "fix_dispatch_failed"
			result.Detail = err.Error()
		} else {
			result.Action = "fix_dispatched"
		}
	}

	return result
}

// _ = s.pushAndMerge(ctx, repoDir, "go-io")
func (s *PrepSubsystem) pushAndMerge(ctx context.Context, repoDir, repo string) error {
	process := s.Core().Process()
	if r := process.RunIn(ctx, repoDir, "git", "push", "github", "HEAD:refs/heads/dev", "--force"); !r.OK {
		return core.E("pushAndMerge", core.Concat("push failed: ", r.Value.(string)), nil)
	}

	process.RunIn(ctx, repoDir, "gh", "pr", "ready", "--repo", core.Concat(GitHubOrg(), "/", repo))

	if r := process.RunIn(ctx, repoDir, "gh", "pr", "merge", "--merge", "--delete-branch"); !r.OK {
		return core.E("pushAndMerge", core.Concat("merge failed: ", r.Value.(string)), nil)
	}

	return nil
}

// _ = s.dispatchFixFromQueue(ctx, "go-io", task)
func (s *PrepSubsystem) dispatchFixFromQueue(ctx context.Context, repo, task string) error {
	input := DispatchInput{
		Repo:  repo,
		Task:  task,
		Agent: "claude:opus",
	}
	_, out, err := s.dispatch(ctx, nil, input)
	if err != nil {
		return err
	}
	if !out.Success {
		return core.E("dispatchFixFromQueue", core.Concat("dispatch failed for ", repo), nil)
	}
	return nil
}

// findings := countFindings(output)
func countFindings(output string) int {
	count := 0
	for _, line := range core.Split(output, "\n") {
		trimmed := core.Trim(line)
		if core.HasPrefix(trimmed, "- ") || core.HasPrefix(trimmed, "* ") ||
			core.Contains(trimmed, "Issue:") || core.Contains(trimmed, "Finding:") ||
			core.Contains(trimmed, "⚠") || core.Contains(trimmed, "❌") {
			count++
		}
	}
	if count == 0 && !core.Contains(output, "No findings") {
		count = 1
	}
	return count
}

// delay := parseRetryAfter("please try after 4 minutes and 56 seconds")
func parseRetryAfter(message string) time.Duration {
	if retryAfterPattern == nil {
		return 5 * time.Minute
	}
	matches := retryAfterPattern.FindStringSubmatch(message)
	if len(matches) >= 2 {
		mins := parseInt(matches[1])
		secs := 0
		if len(matches) >= 3 && matches[2] != "" {
			secs = parseInt(matches[2])
		}
		return time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	}
	return 5 * time.Minute
}

// cmd, args := s.buildReviewCommand(repoDir, "coderabbit")
func (s *PrepSubsystem) buildReviewCommand(repoDir, reviewer string) (string, []string) {
	switch reviewer {
	case "codex":
		return "codex", []string{"review", "--base", "github/main"}
	default:
		return "coderabbit", []string{"review", "--plain", "--base", "github/main", "--config", "CLAUDE.md", "--cwd", repoDir}
	}
}

// s.storeReviewOutput(repoDir, "go-io", "coderabbit", output)
func (s *PrepSubsystem) storeReviewOutput(repoDir, repo, reviewer, output string) {
	dataDir := core.JoinPath(HomeDir(), ".core", "training", "reviews")
	fs.EnsureDir(dataDir)

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := core.Sprintf("%s_%s_%s.txt", repo, reviewer, timestamp)

	fs.Write(core.JoinPath(dataDir, filename), output)

	entry := map[string]string{
		"repo":      repo,
		"reviewer":  reviewer,
		"timestamp": time.Now().Format(time.RFC3339),
		"output":    output,
		"verdict":   "clean",
	}
	if !core.Contains(output, "No findings") && !core.Contains(output, "no issues") {
		entry["verdict"] = "findings"
	}
	jsonLine := core.JSONMarshalString(entry)

	jsonlPath := core.JoinPath(dataDir, "reviews.jsonl")
	r := fs.Append(jsonlPath)
	if !r.OK {
		return
	}
	core.WriteAll(r.Value, core.Concat(jsonLine, "\n"))
}

// s.saveRateLimitState(&RateLimitInfo{Limited: true, RetryAt: time.Now().Add(30 * time.Minute)})
func (s *PrepSubsystem) saveRateLimitState(info *RateLimitInfo) {
	path := core.JoinPath(HomeDir(), ".core", "coderabbit-ratelimit.json")
	if r := fs.WriteAtomic(path, core.JSONMarshalString(info)); !r.OK {
		if err, ok := r.Value.(error); ok {
			core.Warn("reviewQueue: failed to persist rate limit state", "path", path, "reason", err)
			return
		}
		core.Warn("reviewQueue: failed to persist rate limit state", "path", path)
	}
}

// info := s.loadRateLimitState()
func (s *PrepSubsystem) loadRateLimitState() *RateLimitInfo {
	path := core.JoinPath(HomeDir(), ".core", "coderabbit-ratelimit.json")
	r := fs.Read(path)
	if !r.OK {
		return nil
	}
	var info RateLimitInfo
	if ur := core.JSONUnmarshalString(r.Value.(string), &info); !ur.OK {
		return nil
	}
	return &info
}
