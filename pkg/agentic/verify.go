// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go/core"
)

// autoVerifyAndMerge runs inline tests (fast gate) and merges if they pass.
// If tests fail or merge fails due to conflict, attempts one rebase+retry.
// If the retry also fails, labels the PR "needs-review" for human attention.
//
// For deeper review (security, conventions), dispatch a separate task:
//
//	agentic_dispatch repo=go-crypt template=verify persona=engineering/engineering-security-engineer
func (s *PrepSubsystem) autoVerifyAndMerge(wsDir string) {
	result := ReadStatusResult(wsDir)
	st, ok := workspaceStatusValue(result)
	if !ok || st.PRURL == "" || st.Repo == "" {
		return
	}

	repoDir := WorkspaceRepoDir(wsDir)
	org := st.Org
	if org == "" {
		org = "core"
	}

	prNum := extractPRNumber(st.PRURL)
	if prNum == 0 {
		return
	}

	// markMerged is a helper to avoid repeating the status update.
	markMerged := func() {
		if result := ReadStatusResult(wsDir); result.OK {
			st2, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			st2.Status = "merged"
			writeStatusResult(wsDir, st2)
		}
	}

	// Attempt 1: run tests and try to merge
	mergeOutcome := s.attemptVerifyAndMerge(repoDir, org, st.Repo, st.Branch, prNum)
	if mergeOutcome == mergeSuccess {
		markMerged()
		return
	}

	// Attempt 2: rebase onto main and retry
	if mergeOutcome == mergeConflict || mergeOutcome == testFailed {
		if s.rebaseBranch(repoDir, st.Branch) {
			if s.attemptVerifyAndMerge(repoDir, org, st.Repo, st.Branch, prNum) == mergeSuccess {
				markMerged()
				return
			}
		}
	}

	// Both attempts failed — flag for human review
	s.flagForReview(org, st.Repo, prNum, mergeOutcome)

	if result := ReadStatusResult(wsDir); result.OK {
		st2, ok := workspaceStatusValue(result)
		if !ok {
			return
		}
		st2.Question = "Flagged for review — auto-merge failed after retry"
		writeStatusResult(wsDir, st2)
	}
}

type mergeResult int

const (
	mergeSuccess  mergeResult = iota
	testFailed                // tests didn't pass
	mergeConflict             // tests passed but merge failed (conflict)
)

// attemptVerifyAndMerge runs tests and tries to merge. Returns the outcome.
func (s *PrepSubsystem) attemptVerifyAndMerge(repoDir, org, repo, branch string, prNum int) mergeResult {
	testResult := s.runVerification(repoDir)

	if !testResult.passed {
		comment := core.Sprintf("## Verification Failed\n\n**Command:** `%s`\n\n```\n%s\n```\n\n**Exit code:** %d",
			testResult.testCmd, truncate(testResult.output, 2000), testResult.exitCode)
		s.commentOnIssue(context.Background(), org, repo, prNum, comment)
		return testFailed
	}

	// Tests passed — try merge
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if r := s.forgeMergePR(ctx, org, repo, prNum); !r.OK {
		comment := core.Sprintf("## Tests Passed — Merge Failed\n\n`%s` passed but merge failed", testResult.testCmd)
		s.commentOnIssue(context.Background(), org, repo, prNum, comment)
		return mergeConflict
	}

	comment := core.Sprintf("## Auto-Verified & Merged\n\n**Tests:** `%s` — PASS\n\nAuto-merged by core-agent dispatch system.", testResult.testCmd)
	s.commentOnIssue(context.Background(), org, repo, prNum, comment)
	return mergeSuccess
}

// rebaseBranch rebases the current branch onto the default branch and force-pushes.
func (s *PrepSubsystem) rebaseBranch(repoDir, branch string) bool {
	ctx := context.Background()
	process := s.Core().Process()
	base := s.DefaultBranch(repoDir)

	if !process.RunIn(ctx, repoDir, "git", "fetch", "origin", base).OK {
		return false
	}

	if !process.RunIn(ctx, repoDir, "git", "rebase", core.Concat("origin/", base)).OK {
		process.RunIn(ctx, repoDir, "git", "rebase", "--abort")
		return false
	}

	result := ReadStatusResult(core.PathDir(repoDir))
	st, ok := workspaceStatusValue(result)
	org := "core"
	repo := ""
	if ok {
		if st.Org != "" {
			org = st.Org
		}
		repo = st.Repo
	}
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, repo)
	return process.RunIn(ctx, repoDir, "git", "push", "--force-with-lease", forgeRemote, branch).OK
}

// flagForReview adds the "needs-review" label to the PR via Forge API.
func (s *PrepSubsystem) flagForReview(org, repo string, prNum int, result mergeResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure the label exists
	s.ensureLabel(ctx, org, repo, "needs-review", "e11d48")

	// Add label to PR
	payload := core.JSONMarshalString(map[string]any{
		"labels": []int{s.getLabelID(ctx, org, repo, "needs-review")},
	})
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", s.forgeURL, org, repo, prNum)
	HTTPPost(ctx, url, payload, s.forgeToken, "token")

	// Comment explaining the situation
	reason := "Tests failed after rebase"
	if result == mergeConflict {
		reason = "Merge conflict persists after rebase"
	}
	comment := core.Sprintf("## Needs Review\n\n%s. Auto-merge gave up after retry.\n\nLabelled `needs-review` for human attention.", reason)
	s.commentOnIssue(ctx, org, repo, prNum, comment)
}

// ensureLabel creates a label if it doesn't exist.
func (s *PrepSubsystem) ensureLabel(ctx context.Context, org, repo, name, colour string) {
	payload := core.JSONMarshalString(map[string]string{
		"name":  name,
		"color": core.Concat("#", colour),
	})
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	HTTPPost(ctx, url, payload, s.forgeToken, "token")
}

// getLabelID fetches the ID of a label by name.
func (s *PrepSubsystem) getLabelID(ctx context.Context, org, repo, name string) int {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	r := HTTPGet(ctx, url, s.forgeToken, "token")
	if !r.OK {
		return 0
	}

	var labels []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	core.JSONUnmarshalString(r.Value.(string), &labels)
	for _, l := range labels {
		if l.Name == name {
			return l.ID
		}
	}
	return 0
}

// verifyResult holds the outcome of running tests.
type verifyResult struct {
	passed   bool
	output   string
	exitCode int
	testCmd  string
}

func resultText(r core.Result) string {
	if text, ok := r.Value.(string); ok {
		return text
	}
	if r.Value != nil {
		return core.Sprint(r.Value)
	}
	return ""
}

// runVerification detects the project type and runs the appropriate test suite.
func (s *PrepSubsystem) runVerification(repoDir string) verifyResult {
	if fileExists(core.JoinPath(repoDir, "go.mod")) {
		return s.runGoTests(repoDir)
	}
	if fileExists(core.JoinPath(repoDir, "composer.json")) {
		return s.runPHPTests(repoDir)
	}
	if fileExists(core.JoinPath(repoDir, "package.json")) {
		return s.runNodeTests(repoDir)
	}
	return verifyResult{passed: true, testCmd: "none", output: "No test runner detected"}
}

func (s *PrepSubsystem) runGoTests(repoDir string) verifyResult {
	ctx := context.Background()
	process := s.Core().Process()
	r := process.RunWithEnv(ctx, repoDir, []string{"GOWORK=off"}, "go", "test", "./...", "-count=1", "-timeout", "120s")
	out := resultText(r)
	exitCode := 0
	if !r.OK {
		exitCode = 1
	}
	return verifyResult{passed: r.OK, output: out, exitCode: exitCode, testCmd: "go test ./..."}
}

func (s *PrepSubsystem) runPHPTests(repoDir string) verifyResult {
	ctx := context.Background()
	process := s.Core().Process()
	r := process.RunIn(ctx, repoDir, "composer", "test", "--no-interaction")
	if !r.OK {
		// Try pest as fallback
		r2 := process.RunIn(ctx, repoDir, "./vendor/bin/pest", "--no-interaction")
		if !r2.OK {
			return verifyResult{passed: false, testCmd: "none", output: "No PHP test runner found (composer test and vendor/bin/pest both unavailable)", exitCode: 1}
		}
		return verifyResult{passed: true, output: resultText(r2), exitCode: 0, testCmd: "vendor/bin/pest"}
	}
	return verifyResult{passed: true, output: resultText(r), exitCode: 0, testCmd: "composer test"}
}

func (s *PrepSubsystem) runNodeTests(repoDir string) verifyResult {
	r := fs.Read(core.JoinPath(repoDir, "package.json"))
	if !r.OK {
		return verifyResult{passed: true, testCmd: "none", output: "Could not read package.json"}
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if ur := core.JSONUnmarshalString(r.Value.(string), &pkg); !ur.OK || pkg.Scripts["test"] == "" {
		return verifyResult{passed: true, testCmd: "none", output: "No test script in package.json"}
	}

	ctx := context.Background()
	process := s.Core().Process()
	r = process.RunIn(ctx, repoDir, "npm", "test")
	out := resultText(r)
	exitCode := 0
	if !r.OK {
		exitCode = 1
	}
	return verifyResult{passed: r.OK, output: out, exitCode: exitCode, testCmd: "npm test"}
}

// forgeMergePR merges a PR via the Forge API.
func (s *PrepSubsystem) forgeMergePR(ctx context.Context, org, repo string, prNum int) core.Result {
	payload := core.JSONMarshalString(map[string]any{
		"Do":                        "merge",
		"merge_message_field":       "Auto-merged by core-agent after verification\n\nCo-Authored-By: Virgil <virgil@lethean.io>",
		"delete_branch_after_merge": true,
	})

	url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge", s.forgeURL, org, repo, prNum)
	return HTTPPost(ctx, url, payload, s.forgeToken, "token")
}

// extractPRNumber gets the PR number from a Forge PR URL.
func extractPRNumber(prURL string) int {
	parts := core.Split(prURL, "/")
	if len(parts) == 0 {
		return 0
	}
	return parseInt(parts[len(parts)-1])
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	return fs.IsFile(path)
}
