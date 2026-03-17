// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- agentic_review_queue tool ---

// ReviewQueueInput controls the review queue runner.
type ReviewQueueInput struct {
	Limit    int    `json:"limit,omitempty"`    // Max PRs to process this run (default: 4)
	Reviewer string `json:"reviewer,omitempty"` // "coderabbit" (default), "codex", or "both"
	DryRun   bool   `json:"dry_run,omitempty"`  // Preview without acting
	LocalOnly bool  `json:"local_only,omitempty"` // Run review locally, don't touch GitHub
}

// ReviewQueueOutput reports what happened.
type ReviewQueueOutput struct {
	Success   bool           `json:"success"`
	Processed []ReviewResult `json:"processed"`
	Skipped   []string       `json:"skipped,omitempty"`
	RateLimit *RateLimitInfo `json:"rate_limit,omitempty"`
}

// ReviewResult is the outcome of reviewing one repo.
type ReviewResult struct {
	Repo     string `json:"repo"`
	Verdict  string `json:"verdict"`  // clean, findings, rate_limited, error
	Findings int    `json:"findings"` // Number of findings (0 = clean)
	Action   string `json:"action"`   // merged, fix_dispatched, skipped, waiting
	Detail   string `json:"detail,omitempty"`
}

// RateLimitInfo tracks CodeRabbit rate limit state.
type RateLimitInfo struct {
	Limited  bool      `json:"limited"`
	RetryAt  time.Time `json:"retry_at,omitempty"`
	Message  string    `json:"message,omitempty"`
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

	basePath := filepath.Join(s.codePath, "core")

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
			skipped = append(skipped, repo+" (limit reached)")
			continue
		}

		// Check rate limit from previous run
		if rateInfo != nil && rateInfo.Limited && time.Now().Before(rateInfo.RetryAt) {
			skipped = append(skipped, repo+" (rate limited)")
			continue
		}

		repoDir := filepath.Join(basePath, repo)
		result := s.reviewRepo(ctx, repoDir, repo, input.DryRun, input.LocalOnly)

		// Parse rate limit from result
		if result.Verdict == "rate_limited" {
			retryAfter := parseRetryAfter(result.Detail)
			rateInfo = &RateLimitInfo{
				Limited: true,
				RetryAt: time.Now().Add(retryAfter),
				Message: result.Detail,
			}
			// Don't count rate-limited as processed — save the slot
			skipped = append(skipped, repo+" (rate limited: "+retryAfter.String()+")")
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
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil
	}

	var candidates []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		repoDir := filepath.Join(basePath, e.Name())
		if !hasRemote(repoDir, "github") {
			continue
		}
		ahead := commitsAhead(repoDir, "github/main", "HEAD")
		if ahead > 0 {
			candidates = append(candidates, e.Name())
		}
	}
	return candidates
}

// reviewRepo runs CodeRabbit on a single repo and takes action.
func (s *PrepSubsystem) reviewRepo(ctx context.Context, repoDir, repo string, dryRun, localOnly bool) ReviewResult {
	result := ReviewResult{Repo: repo}

	// Check saved rate limit
	if rl := s.loadRateLimitState(); rl != nil && rl.Limited && time.Now().Before(rl.RetryAt) {
		result.Verdict = "rate_limited"
		result.Detail = fmt.Sprintf("retry after %s", rl.RetryAt.Format(time.RFC3339))
		return result
	}

	// Run reviewer CLI locally
	reviewer := "coderabbit" // default, can be overridden by caller
	cmd := s.buildReviewCommand(ctx, repoDir, reviewer)
	out, err := cmd.CombinedOutput()
	output := string(out)

	// Parse rate limit (both reviewers use similar patterns)
	if strings.Contains(output, "Rate limit exceeded") || strings.Contains(output, "rate limit") {
		result.Verdict = "rate_limited"
		result.Detail = output
		return result
	}

	// Parse error
	if err != nil && !strings.Contains(output, "No findings") && !strings.Contains(output, "no issues") {
		result.Verdict = "error"
		result.Detail = output
		return result
	}

	// Store raw output for training data
	s.storeReviewOutput(repoDir, repo, reviewer, output)

	// Parse verdict
	if strings.Contains(output, "No findings") || strings.Contains(output, "no issues") || strings.Contains(output, "LGTM") {
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
			result.Action = "push failed: " + err.Error()
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
		findingsFile := filepath.Join(repoDir, ".core", "coderabbit-findings.txt")
		coreio.Local.Write(findingsFile, output)

		// Dispatch fix agent with the findings
		task := fmt.Sprintf("Fix CodeRabbit findings. The review output is in .core/coderabbit-findings.txt. "+
			"Read it, verify each finding against the code, fix what's valid. Run tests. "+
			"Commit: fix(coderabbit): address review findings\n\nFindings summary (%d issues):\n%s",
			result.Findings, truncate(output, 1500))

		s.dispatchFixFromQueue(ctx, repo, task)
		result.Action = "fix_dispatched"
	}

	return result
}

// pushAndMerge pushes to GitHub dev and merges the PR.
func (s *PrepSubsystem) pushAndMerge(ctx context.Context, repoDir, repo string) error {
	// Push to dev
	pushCmd := exec.CommandContext(ctx, "git", "push", "github", "HEAD:refs/heads/dev", "--force")
	pushCmd.Dir = repoDir
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return coreerr.E("pushAndMerge", "push failed: "+string(out), err)
	}

	// Mark PR ready if draft
	readyCmd := exec.CommandContext(ctx, "gh", "pr", "ready", "--repo", GitHubOrg()+"/"+repo)
	readyCmd.Dir = repoDir
	readyCmd.Run() // Ignore error — might already be ready

	// Try to merge
	mergeCmd := exec.CommandContext(ctx, "gh", "pr", "merge", "--merge", "--delete-branch")
	mergeCmd.Dir = repoDir
	if out, err := mergeCmd.CombinedOutput(); err != nil {
		return coreerr.E("pushAndMerge", "merge failed: "+string(out), err)
	}

	return nil
}

// dispatchFixFromQueue dispatches an opus agent to fix CodeRabbit findings.
func (s *PrepSubsystem) dispatchFixFromQueue(ctx context.Context, repo, task string) {
	// Use the dispatch system — creates workspace, spawns agent
	input := DispatchInput{
		Repo:  repo,
		Task:  task,
		Agent: "claude:opus",
	}
	s.dispatch(ctx, nil, input)
}

// countFindings estimates the number of findings in CodeRabbit output.
func countFindings(output string) int {
	// Count lines that look like findings
	count := 0
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
			strings.Contains(trimmed, "Issue:") || strings.Contains(trimmed, "Finding:") ||
			strings.Contains(trimmed, "⚠") || strings.Contains(trimmed, "❌") {
			count++
		}
	}
	if count == 0 && !strings.Contains(output, "No findings") {
		count = 1 // At least one finding if not clean
	}
	return count
}

// parseRetryAfter extracts the retry duration from a rate limit message.
// Example: "please try after 4 minutes and 56 seconds"
func parseRetryAfter(message string) time.Duration {
	re := regexp.MustCompile(`(\d+)\s*minutes?\s*(?:and\s*)?(\d+)?\s*seconds?`)
	matches := re.FindStringSubmatch(message)
	if len(matches) >= 2 {
		mins, _ := strconv.Atoi(matches[1])
		secs := 0
		if len(matches) >= 3 && matches[2] != "" {
			secs, _ = strconv.Atoi(matches[2])
		}
		return time.Duration(mins)*time.Minute + time.Duration(secs)*time.Second
	}
	// Default: 5 minutes
	return 5 * time.Minute
}

// buildReviewCommand creates the CLI command for the chosen reviewer.
func (s *PrepSubsystem) buildReviewCommand(ctx context.Context, repoDir, reviewer string) *exec.Cmd {
	switch reviewer {
	case "codex":
		return exec.CommandContext(ctx, "codex", "review", "--base", "github/main")
	default: // coderabbit
		return exec.CommandContext(ctx, "coderabbit", "review", "--plain",
			"--base", "github/main", "--config", "CLAUDE.md", "--cwd", repoDir)
	}
}

// storeReviewOutput saves raw review output for training data collection.
func (s *PrepSubsystem) storeReviewOutput(repoDir, repo, reviewer, output string) {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".core", "training", "reviews")
	coreio.Local.EnsureDir(dataDir)

	timestamp := time.Now().Format("2006-01-02T15-04-05")
	filename := fmt.Sprintf("%s_%s_%s.txt", repo, reviewer, timestamp)

	// Write raw output
	coreio.Local.Write(filepath.Join(dataDir, filename), output)

	// Append to JSONL for structured training
	entry := map[string]string{
		"repo":      repo,
		"reviewer":  reviewer,
		"timestamp": time.Now().Format(time.RFC3339),
		"output":    output,
		"verdict":   "clean",
	}
	if !strings.Contains(output, "No findings") && !strings.Contains(output, "no issues") {
		entry["verdict"] = "findings"
	}
	jsonLine, _ := json.Marshal(entry)

	jsonlPath := filepath.Join(dataDir, "reviews.jsonl")
	f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		defer f.Close()
		f.Write(append(jsonLine, '\n'))
	}
}

// saveRateLimitState persists rate limit info for cross-run awareness.
func (s *PrepSubsystem) saveRateLimitState(info *RateLimitInfo) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".core", "coderabbit-ratelimit.json")
	data, _ := json.Marshal(info)
	coreio.Local.Write(path, string(data))
}

// loadRateLimitState reads persisted rate limit info.
func (s *PrepSubsystem) loadRateLimitState() *RateLimitInfo {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".core", "coderabbit-ratelimit.json")
	data, err := coreio.Local.Read(path)
	if err != nil {
		return nil
	}
	var info RateLimitInfo
	if json.Unmarshal([]byte(data), &info) != nil {
		return nil
	}
	return &info
}
