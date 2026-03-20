// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
)

// autoVerifyAndMerge runs inline tests (fast gate) and merges if they pass.
// If tests fail or merge fails due to conflict, attempts one rebase+retry.
// If the retry also fails, labels the PR "needs-review" for human attention.
//
// For deeper review (security, conventions), dispatch a separate task:
//
//	agentic_dispatch repo=go-crypt template=verify persona=engineering/engineering-security-engineer
func (s *PrepSubsystem) autoVerifyAndMerge(wsDir string) {
	st, err := readStatus(wsDir)
	if err != nil || st.PRURL == "" || st.Repo == "" {
		return
	}

	srcDir := filepath.Join(wsDir, "src")
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
		if st2, err := readStatus(wsDir); err == nil {
			st2.Status = "merged"
			writeStatus(wsDir, st2)
		}
	}

	// Attempt 1: run tests and try to merge
	result := s.attemptVerifyAndMerge(srcDir, org, st.Repo, st.Branch, prNum)
	if result == mergeSuccess {
		markMerged()
		return
	}

	// Attempt 2: rebase onto main and retry
	if result == mergeConflict || result == testFailed {
		if s.rebaseBranch(srcDir, st.Branch) {
			if s.attemptVerifyAndMerge(srcDir, org, st.Repo, st.Branch, prNum) == mergeSuccess {
				markMerged()
				return
			}
		}
	}

	// Both attempts failed — flag for human review
	s.flagForReview(org, st.Repo, prNum, result)

	if st2, err := readStatus(wsDir); err == nil {
		st2.Question = "Flagged for review — auto-merge failed after retry"
		writeStatus(wsDir, st2)
	}
}

type mergeResult int

const (
	mergeSuccess  mergeResult = iota
	testFailed                // tests didn't pass
	mergeConflict             // tests passed but merge failed (conflict)
)

// attemptVerifyAndMerge runs tests and tries to merge. Returns the outcome.
func (s *PrepSubsystem) attemptVerifyAndMerge(srcDir, org, repo, branch string, prNum int) mergeResult {
	testResult := s.runVerification(srcDir)

	if !testResult.passed {
		comment := fmt.Sprintf("## Verification Failed\n\n**Command:** `%s`\n\n```\n%s\n```\n\n**Exit code:** %d",
			testResult.testCmd, truncate(testResult.output, 2000), testResult.exitCode)
		s.commentOnIssue(context.Background(), org, repo, prNum, comment)
		return testFailed
	}

	// Tests passed — try merge
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.forgeMergePR(ctx, org, repo, prNum); err != nil {
		comment := fmt.Sprintf("## Tests Passed — Merge Failed\n\n`%s` passed but merge failed: %v", testResult.testCmd, err)
		s.commentOnIssue(context.Background(), org, repo, prNum, comment)
		return mergeConflict
	}

	comment := fmt.Sprintf("## Auto-Verified & Merged\n\n**Tests:** `%s` — PASS\n\nAuto-merged by core-agent dispatch system.", testResult.testCmd)
	s.commentOnIssue(context.Background(), org, repo, prNum, comment)
	return mergeSuccess
}

// rebaseBranch rebases the current branch onto origin/main and force-pushes.
func (s *PrepSubsystem) rebaseBranch(srcDir, branch string) bool {
	// Fetch latest main
	fetch := exec.Command("git", "fetch", "origin", "main")
	fetch.Dir = srcDir
	if err := fetch.Run(); err != nil {
		return false
	}

	// Rebase onto main
	rebase := exec.Command("git", "rebase", "origin/main")
	rebase.Dir = srcDir
	if err := rebase.Run(); err != nil {
		// Rebase failed — abort and give up
		abort := exec.Command("git", "rebase", "--abort")
		abort.Dir = srcDir
		abort.Run()
		return false
	}

	// Force-push the rebased branch
	push := exec.Command("git", "push", "--force-with-lease", "origin", branch)
	push.Dir = srcDir
	return push.Run() == nil
}

// flagForReview adds the "needs-review" label to the PR via Forge API.
func (s *PrepSubsystem) flagForReview(org, repo string, prNum int, result mergeResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Ensure the label exists
	s.ensureLabel(ctx, org, repo, "needs-review", "e11d48")

	// Add label to PR
	payload, _ := json.Marshal(map[string]any{
		"labels": []int{s.getLabelID(ctx, org, repo, "needs-review")},
	})
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", s.forgeURL, org, repo, prNum)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)
	resp, err := s.client.Do(req)
	if err == nil {
		resp.Body.Close()
	}

	// Comment explaining the situation
	reason := "Tests failed after rebase"
	if result == mergeConflict {
		reason = "Merge conflict persists after rebase"
	}
	comment := fmt.Sprintf("## Needs Review\n\n%s. Auto-merge gave up after retry.\n\nLabelled `needs-review` for human attention.", reason)
	s.commentOnIssue(ctx, org, repo, prNum, comment)
}

// ensureLabel creates a label if it doesn't exist.
func (s *PrepSubsystem) ensureLabel(ctx context.Context, org, repo, name, colour string) {
	payload, _ := json.Marshal(map[string]string{
		"name":  name,
		"color": "#" + colour,
	})
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)
	resp, err := s.client.Do(req)
	if err == nil {
		resp.Body.Close()
	}
}

// getLabelID fetches the ID of a label by name.
func (s *PrepSubsystem) getLabelID(ctx context.Context, org, repo, name string) int {
	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "token "+s.forgeToken)
	resp, err := s.client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var labels []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	json.NewDecoder(resp.Body).Decode(&labels)
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

// runVerification detects the project type and runs the appropriate test suite.
func (s *PrepSubsystem) runVerification(srcDir string) verifyResult {
	if fileExists(filepath.Join(srcDir, "go.mod")) {
		return s.runGoTests(srcDir)
	}
	if fileExists(filepath.Join(srcDir, "composer.json")) {
		return s.runPHPTests(srcDir)
	}
	if fileExists(filepath.Join(srcDir, "package.json")) {
		return s.runNodeTests(srcDir)
	}
	return verifyResult{passed: true, testCmd: "none", output: "No test runner detected"}
}

func (s *PrepSubsystem) runGoTests(srcDir string) verifyResult {
	cmd := exec.Command("go", "test", "./...", "-count=1", "-timeout", "120s")
	cmd.Dir = srcDir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return verifyResult{passed: exitCode == 0, output: string(out), exitCode: exitCode, testCmd: "go test ./..."}
}

func (s *PrepSubsystem) runPHPTests(srcDir string) verifyResult {
	cmd := exec.Command("composer", "test", "--no-interaction")
	cmd.Dir = srcDir
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			cmd2 := exec.Command("./vendor/bin/pest", "--no-interaction")
			cmd2.Dir = srcDir
			out2, err2 := cmd2.CombinedOutput()
			if err2 != nil {
				return verifyResult{passed: true, testCmd: "none", output: "No PHP test runner found"}
			}
			return verifyResult{passed: true, output: string(out2), exitCode: 0, testCmd: "vendor/bin/pest"}
		}
	}

	return verifyResult{passed: exitCode == 0, output: string(out), exitCode: exitCode, testCmd: "composer test"}
}

func (s *PrepSubsystem) runNodeTests(srcDir string) verifyResult {
	data, err := coreio.Local.Read(filepath.Join(srcDir, "package.json"))
	if err != nil {
		return verifyResult{passed: true, testCmd: "none", output: "Could not read package.json"}
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal([]byte(data), &pkg) != nil || pkg.Scripts["test"] == "" {
		return verifyResult{passed: true, testCmd: "none", output: "No test script in package.json"}
	}

	cmd := exec.Command("npm", "test")
	cmd.Dir = srcDir
	out, err := cmd.CombinedOutput()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return verifyResult{passed: exitCode == 0, output: string(out), exitCode: exitCode, testCmd: "npm test"}
}

// forgeMergePR merges a PR via the Forge API.
func (s *PrepSubsystem) forgeMergePR(ctx context.Context, org, repo string, prNum int) error {
	payload, _ := json.Marshal(map[string]any{
		"Do":                        "merge",
		"merge_message_field":       "Auto-merged by core-agent after verification\n\nCo-Authored-By: Virgil <virgil@lethean.io>",
		"delete_branch_after_merge": true,
	})

	url := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge", s.forgeURL, org, repo, prNum)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "token "+s.forgeToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return coreerr.E("forgeMergePR", "request failed", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		var errBody map[string]any
		json.NewDecoder(resp.Body).Decode(&errBody)
		msg, _ := errBody["message"].(string)
		return coreerr.E("forgeMergePR", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, msg), nil)
	}

	return nil
}

// extractPRNumber gets the PR number from a Forge PR URL.
func extractPRNumber(prURL string) int {
	parts := strings.Split(prURL, "/")
	if len(parts) == 0 {
		return 0
	}
	var num int
	fmt.Sscanf(parts[len(parts)-1], "%d", &num)
	return num
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	return coreio.Local.IsFile(path)
}
