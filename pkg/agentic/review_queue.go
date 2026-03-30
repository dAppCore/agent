// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"regexp"
	"time"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_review_queue tool ---

// ReviewQueueInput controls the review queue runner.
//
//	input := agentic.ReviewQueueInput{Reviewer: "coderabbit", Limit: 4, DryRun: true}
type ReviewQueueInput struct {
	Limit     int    `json:"limit,omitempty"`      // Max PRs to process this run (default: 4)
	Reviewer  string `json:"reviewer,omitempty"`   // "coderabbit" (default), "codex", or "both"
	DryRun    bool   `json:"dry_run,omitempty"`    // Preview without acting
	LocalOnly bool   `json:"local_only,omitempty"` // Run review locally, don't touch GitHub
}

// ReviewQueueOutput reports what happened.
//
//	out := agentic.ReviewQueueOutput{Success: true, Processed: []agentic.ReviewResult{{Repo: "go-io", Verdict: "clean"}}}
type ReviewQueueOutput struct {
	Success   bool           `json:"success"`
	Processed []ReviewResult `json:"processed"`
	Skipped   []string       `json:"skipped,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
}

// ReviewResult is the outcome of reviewing one repo.
//
//	result := agentic.ReviewResult{Repo: "go-io", Verdict: "findings", Findings: 3, Action: "fix_dispatched"}
type ReviewResult struct {
	Repo     string `json:"repo"`
	Verdict  string `json:"verdict"`  // clean, findings, rate_limited, error
	Findings int    `json:"findings"` // Number of findings (0 = clean)
	Action   string `json:"action"`   // merged, fix_dispatched, skipped, waiting
	Detail   string `json:"detail,omitempty"`
}

// RateLimitInfo tracks CodeRabbit rate limit state.
//
//	limit := agentic.RateLimitInfo{Limited: true, Message: "retry after 2026-03-22T06:00:00Z"}
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
		Description: "Process the CodeRabbit review queue. Runs local CodeRabbit review on repos, auto-merges clean ones on GitHub, dispatches fix agents for findings. Respects rate limits.",
	}, s.reviewQueue)
}

func (s *PrepSubsystem) reviewQueue(ctx context.Context, _ *mcp.CallToolRequest, input ReviewQueueInput) (*mcp.CallToolResult, ReviewQueueOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 4
	}

	basePath := core.JoinPath(s.codePath, "core")

	// Find repos with draft PRs (ahead of GitHub)
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
		if len(processed) >= limit {
			skipped = append(skipped, core.Concat(repo, " (limit reached)"))
			continue
		}

		// Check rate limit from previous run
		if rateInfo != nil && rateInfo.Limited && time.Now().Before(rateInfo.RetryAt) {
			skipped = append(skipped, core.Concat(repo, " (rate limited)"))
			continue
		}

		repoDir := core.JoinPath(basePath, repo)
		reviewer := input.Reviewer
		if reviewer == "" {
			reviewer = "coderabbit"
		}
		result := s.reviewRepo(ctx, repoDir, repo, reviewer, input.DryRun, input.LocalOnly)

		// Parse rate limit from result
		if result.Verdict == "rate_limited" {
			retryAfter := parseRetryAfter(result.Detail)
			rateInfo = &RateLimitInfo{
				Limited: true,
				RetryAt: time.Now().Add(retryAfter),
				Message: result.Detail,
			}
			// Don't count rate-limited as processed — save the slot
			skipped = append(skipped, core.Concat(repo, " (rate limited: ", retryAfter.String(), ")"))
			continue
		}

		processed = append(processed, result)
	}

	// Save rate limit state for next run
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

// findReviewCandidates returns repos that are ahead of GitHub main.
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

// reviewRepo runs CodeRabbit on a single repo and takes action.
func (s *PrepSubsystem) reviewRepo(ctx context.Context, repoDir, repo, reviewer string, dryRun, localOnly bool) ReviewResult {
	result := ReviewResult{Repo: repo}

	// Check saved rate limit
	if rl := s.loadRateLimitState(); rl != nil && rl.Limited && time.Now().Before(rl.RetryAt) {
		result.Verdict = "rate_limited"
		result.Detail = core.Sprintf("retry after %s", rl.RetryAt.Format(time.RFC3339))
		return result
	}

	// Run reviewer CLI locally — use the reviewer passed from reviewQueue
	if reviewer == "" {
		reviewer = "coderabbit"
	}
	command, args := s.buildReviewCommand(repoDir, reviewer)
	r := s.runCmd(ctx, repoDir, command, args...)
	output, _ := r.Value.(string)

	// Parse rate limit (both reviewers use similar patterns)
	if core.Contains(output, "Rate limit exceeded") || core.Contains(output, "rate limit") {
		result.Verdict = "rate_limited"
		result.Detail = output
		return result
	}

	// Parse error
	if !r.OK && !core.Contains(output, "No findings") && !core.Contains(output, "no issues") {
		result.Verdict = "error"
		result.Detail = output
		return result
	}

	// Store raw output for training data
	s.storeReviewOutput(repoDir, repo, reviewer, output)

	// Parse verdict
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

		// Push to GitHub and mark PR ready / merge
		if err := s.pushAndMerge(ctx, repoDir, repo); err != nil {
			result.Action = core.Concat("push failed: ", err.Error())
		} else {
			result.Action = "merged"
		}
	} else {
		// Has findings — count them and dispatch fix agent
		result.Verdict = "findings"
		result.Findings = countFindings(output)
		result.Detail = truncate(output, 500)

		if dryRun {
			result.Action = "skipped (dry run)"
			return result
		}

		// Save findings for agent dispatch
		findingsFile := core.JoinPath(repoDir, ".core", "coderabbit-findings.txt")
		fs.Write(findingsFile, output)

		// Dispatch fix agent with the findings
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

// pushAndMerge pushes to GitHub dev and merges the PR.
func (s *PrepSubsystem) pushAndMerge(ctx context.Context, repoDir, repo string) error {
	if r := s.gitCmd(ctx, repoDir, "push", "github", "HEAD:refs/heads/dev", "--force"); !r.OK {
		return core.E("pushAndMerge", core.Concat("push failed: ", r.Value.(string)), nil)
	}

	// Mark PR ready if draft
	s.runCmdOK(ctx, repoDir, "gh", "pr", "ready", "--repo", core.Concat(GitHubOrg(), "/", repo))

	if r := s.runCmd(ctx, repoDir, "gh", "pr", "merge", "--merge", "--delete-branch"); !r.OK {
		return core.E("pushAndMerge", core.Concat("merge failed: ", r.Value.(string)), nil)
	}

	return nil
}

// dispatchFixFromQueue dispatches an opus agent to fix CodeRabbit findings.
func (s *PrepSubsystem) dispatchFixFromQueue(ctx context.Context, repo, task string) error {
	// Use the dispatch system — creates workspace, spawns agent
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

// countFindings estimates the number of findings in CodeRabbit output.
func countFindings(output string) int {
	// Count lines that look like findings
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
		count = 1 // At least one finding if not clean
	}
	return count
}

// parseRetryAfter extracts the retry duration from a rate limit message.
// Example: "please try after 4 minutes and 56 seconds"
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
	// Default: 5 minutes
	return 5 * time.Minute
}

// buildReviewCommand returns the command and args for the chosen reviewer.
//
//	cmd, args := s.buildReviewCommand(repoDir, "coderabbit")
func (s *PrepSubsystem) buildReviewCommand(repoDir, reviewer string) (string, []string) {
	switch reviewer {
	case "codex":
		return "codex", []string{"review", "--base", "github/main"}
	default: // coderabbit
		return "coderabbit", []string{"review", "--plain", "--base", "github/main", "--config", "CLAUDE.md", "--cwd", repoDir}
	}
}

// storeReviewOutput saves raw review output for training data collection.
func (s *PrepSubsystem) storeReviewOutput(repoDir, repo, reviewer, output string) {
	dataDir := core.JoinPath(core.Env("DIR_HOME"), ".core", "training", "reviews")
	fs.EnsureDir(dataDir)

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := core.Sprintf("%s_%s_%s.txt", repo, reviewer, timestamp)

	// Write raw output
	fs.Write(core.JoinPath(dataDir, filename), output)

	// Append to JSONL for structured training
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

// saveRateLimitState persists rate limit info for cross-run awareness.
func (s *PrepSubsystem) saveRateLimitState(info *RateLimitInfo) {
	path := core.JoinPath(core.Env("DIR_HOME"), ".core", "coderabbit-ratelimit.json")
	fs.Write(path, core.JSONMarshalString(info))
}

// loadRateLimitState reads persisted rate limit info.
func (s *PrepSubsystem) loadRateLimitState() *RateLimitInfo {
	path := core.JoinPath(core.Env("DIR_HOME"), ".core", "coderabbit-ratelimit.json")
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
