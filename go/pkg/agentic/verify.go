// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"time"

	core "dappco.re/go"
)

// s.autoVerifyAndMerge("/srv/core/workspace/core/go-io/task-5")
func (s *PrepSubsystem) autoVerifyAndMerge(workspaceDir string) {
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok || workspaceStatus.PRURL == "" || workspaceStatus.Repo == "" {
		return
	}

	repoDir := WorkspaceRepoDir(workspaceDir)
	org := workspaceStatus.Org
	if org == "" {
		org = "core"
	}

	pullRequestNumber := extractPullRequestNumber(workspaceStatus.PRURL)
	if pullRequestNumber == 0 {
		return
	}

	markMerged := func() {
		if result := ReadStatusResult(workspaceDir); result.OK {
			st2, ok := workspaceStatusValue(result)
			if !ok {
				return
			}
			st2.Status = "merged"
			// Reported: the status file is what the monitor polls, so a silent write
			// failure leaves the workspace looking stuck indefinitely.
			if r := writeStatusResult(workspaceDir, st2); !r.OK {
				core.Warn("agentic: failed to write workspace status", "reason", r.Value)
			}
		}
	}

	mergeOutcome := s.attemptVerifyAndMerge(repoDir, org, workspaceStatus.Repo, workspaceStatus.Branch, pullRequestNumber)
	if mergeOutcome == mergeSuccess {
		markMerged()
		return
	}

	if mergeOutcome == mergeConflict || mergeOutcome == testFailed {
		if s.rebaseBranch(repoDir, workspaceStatus.Branch) {
			if s.attemptVerifyAndMerge(repoDir, org, workspaceStatus.Repo, workspaceStatus.Branch, pullRequestNumber) == mergeSuccess {
				markMerged()
				return
			}
		}
	}

	s.flagForReview(org, workspaceStatus.Repo, pullRequestNumber, mergeOutcome)

	if result := ReadStatusResult(workspaceDir); result.OK {
		workspaceStatusUpdate, ok := workspaceStatusValue(result)
		if !ok {
			return
		}
		workspaceStatusUpdate.Question = "Flagged for review — auto-merge failed after retry"
		// Reported: the status file is what the monitor polls, so a silent write
		// failure leaves the workspace looking stuck indefinitely.
		if r := writeStatusResult(workspaceDir, workspaceStatusUpdate); !r.OK {
			core.Warn("agentic: failed to write workspace status", "reason", r.Value)
		}
	}
}

type mergeResult int

const (
	mergeSuccess mergeResult = iota
	testFailed
	mergeConflict
)

// s.attemptVerifyAndMerge("/srv/core/workspace/core/go-io/task-5/repo", "core", "go-io", "feature/ax-cleanup", 42)
func (s *PrepSubsystem) attemptVerifyAndMerge(repoDir, org, repo, branch string, pullRequestNumber int) mergeResult {
	testResult := s.runVerification(repoDir)

	if !testResult.passed {
		comment := core.Sprintf("## Verification Failed\n\n**Command:** `%s`\n\n```\n%s\n```\n\n**Exit code:** %d",
			testResult.testCmd, truncate(testResult.output, 2000), testResult.exitCode)
		s.commentOnIssue(context.Background(), org, repo, pullRequestNumber, comment)
		return testFailed
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if mergeAttempt := s.forgeMergePR(ctx, org, repo, pullRequestNumber); !mergeAttempt.OK {
		comment := core.Sprintf("## Tests Passed — Merge Failed\n\n`%s` passed but merge failed", testResult.testCmd)
		s.commentOnIssue(context.Background(), org, repo, pullRequestNumber, comment)
		return mergeConflict
	}

	comment := core.Sprintf("## Auto-Verified & Merged\n\n**Tests:** `%s` — PASS\n\nAuto-merged by core-agent dispatch system.", testResult.testCmd)
	s.commentOnIssue(context.Background(), org, repo, pullRequestNumber, comment)
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, repo)
	s.cleanupForgeBranch(ctx, repoDir, forgeRemote, branch)
	return mergeSuccess
}

// s.rebaseBranch("/srv/core/workspace/core/go-io/task-5/repo", "feature/ax-cleanup")
func (s *PrepSubsystem) rebaseBranch(repoDir, branch string) bool {
	ctx := context.Background()
	process := s.Core().Process()
	base := s.DefaultBranch(repoDir)

	if !process.RunIn(ctx, repoDir, "git", "fetch", "origin", base).OK {
		return false
	}

	if !process.RunIn(ctx, repoDir, "git", "rebase", core.Concat("origin/", base)).OK {
		// Best-effort: there may be no rebase in progress to abort.
		_ = process.RunIn(ctx, repoDir, "git", "rebase", "--abort")
		return false
	}

	result := ReadStatusResult(core.PathDir(repoDir))
	workspaceStatus, ok := workspaceStatusValue(result)
	org := "core"
	repo := ""
	if ok {
		if workspaceStatus.Org != "" {
			org = workspaceStatus.Org
		}
		repo = workspaceStatus.Repo
	}
	forgeRemote := core.Sprintf("ssh://git@forge.lthn.ai:2223/%s/%s.git", org, repo)
	return process.RunIn(ctx, repoDir, "git", "push", "--force-with-lease", forgeRemote, branch).OK
}

// s.flagForReview("core", "go-io", 42, mergeConflict)
func (s *PrepSubsystem) flagForReview(org, repo string, pullRequestNumber int, mergeOutcome mergeResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.ensureLabel(ctx, org, repo, "needs-review", "e11d48")

	payload := core.JSONMarshalString(map[string]any{
		"labels": []int{s.getLabelID(ctx, org, repo, "needs-review")},
	})
	url := core.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/labels", s.forgeURL, org, repo, pullRequestNumber)
	// Reported: a label that fails to apply silently drops the PR out of the
	// review queue that selects on it.
	if r := HTTPPost(ctx, url, payload, s.forgeToken, "token"); !r.OK {
		core.Warn("agentic: failed to apply PR label", "url", url, "reason", r.Value)
	}

	reason := "Tests failed after rebase"
	if mergeOutcome == mergeConflict {
		reason = "Merge conflict persists after rebase"
	}
	comment := core.Sprintf("## Needs Review\n\n%s. Auto-merge gave up after retry.\n\nLabelled `needs-review` for human attention.", reason)
	s.commentOnIssue(ctx, org, repo, pullRequestNumber, comment)
}

// s.ensureLabel(context.Background(), "core", "go-io", "needs-review", "e11d48")
func (s *PrepSubsystem) ensureLabel(ctx context.Context, org, repo, name, colour string) {
	payload := core.JSONMarshalString(map[string]string{
		"name":  name,
		"color": core.Concat("#", colour),
	})
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	// Reported: a label that fails to apply silently drops the PR out of the
	// review queue that selects on it.
	if r := HTTPPost(ctx, url, payload, s.forgeToken, "token"); !r.OK {
		core.Warn("agentic: failed to apply PR label", "url", url, "reason", r.Value)
	}
}

// s.getLabelID(context.Background(), "core", "go-io", "needs-review")
func (s *PrepSubsystem) getLabelID(ctx context.Context, org, repo, name string) int {
	url := core.Sprintf("%s/api/v1/repos/%s/%s/labels", s.forgeURL, org, repo)
	getResult := HTTPGet(ctx, url, s.forgeToken, "token")
	if !getResult.OK {
		return 0
	}

	var labels []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	// Reported: unparsable labels means getLabelID returns nothing and the
	// caller silently applies no label at all.
	if r := core.JSONUnmarshalString(getResult.Value.(string), &labels); !r.OK {
		core.Warn("agentic: failed to parse repo labels", "reason", r.Value)
	}
	for _, l := range labels {
		if l.Name == name {
			return l.ID
		}
	}
	return 0
}

type verifyResult struct {
	passed   bool
	output   string
	exitCode int
	testCmd  string
}

func resultText(result core.Result) string {
	if text, ok := result.Value.(string); ok {
		return text
	}
	if result.Value != nil {
		return core.Sprint(result.Value)
	}
	return ""
}

// s.runVerification("/srv/core/workspace/core/go-io/task-5/repo")
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
	processResult := process.RunWithEnv(ctx, repoDir, []string{"GOWORK=off"}, "go", "test", "./...", "-count=1", "-timeout", "120s")
	out := resultText(processResult)
	exitCode := 0
	if !processResult.OK {
		exitCode = 1
	}
	return verifyResult{passed: processResult.OK, output: out, exitCode: exitCode, testCmd: "go test ./..."}
}

func (s *PrepSubsystem) runPHPTests(repoDir string) verifyResult {
	ctx := context.Background()
	process := s.Core().Process()
	composerResult := process.RunIn(ctx, repoDir, "composer", "test", "--no-interaction")
	if !composerResult.OK {
		fallbackResult := process.RunIn(ctx, repoDir, "./vendor/bin/pest", "--no-interaction")
		if !fallbackResult.OK {
			return verifyResult{passed: false, testCmd: "none", output: "No PHP test runner found (composer test and vendor/bin/pest both unavailable)", exitCode: 1}
		}
		return verifyResult{passed: true, output: resultText(fallbackResult), exitCode: 0, testCmd: "vendor/bin/pest"}
	}
	return verifyResult{passed: true, output: resultText(composerResult), exitCode: 0, testCmd: "composer test"}
}

func (s *PrepSubsystem) runNodeTests(repoDir string) verifyResult {
	packageResult := fs.Read(core.JoinPath(repoDir, "package.json"))
	if !packageResult.OK {
		return verifyResult{passed: true, testCmd: "none", output: "Could not read package.json"}
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if parseResult := core.JSONUnmarshalString(packageResult.Value.(string), &pkg); !parseResult.OK || pkg.Scripts["test"] == "" {
		return verifyResult{passed: true, testCmd: "none", output: "No test script in package.json"}
	}

	ctx := context.Background()
	process := s.Core().Process()
	testResult := process.RunIn(ctx, repoDir, "npm", "test")
	out := resultText(testResult)
	exitCode := 0
	if !testResult.OK {
		exitCode = 1
	}
	return verifyResult{passed: testResult.OK, output: out, exitCode: exitCode, testCmd: "npm test"}
}

// s.forgeMergePR(context.Background(), "core", "go-io", 42)
func (s *PrepSubsystem) forgeMergePR(ctx context.Context, org, repo string, pullRequestNumber int) core.Result {
	payload := core.JSONMarshalString(map[string]any{
		"Do":                        "merge",
		"merge_message_field":       "Auto-merged by core-agent after verification\n\nCo-Authored-By: Virgil <virgil@lethean.io>",
		"delete_branch_after_merge": true,
	})

	url := core.Sprintf("%s/api/v1/repos/%s/%s/pulls/%d/merge", s.forgeURL, org, repo, pullRequestNumber)
	return HTTPPost(ctx, url, payload, s.forgeToken, "token")
}

// extractPullRequestNumber("https://forge.lthn.ai/core/go-io/pulls/42")
func extractPullRequestNumber(pullRequestURL string) int {
	parts := core.Split(pullRequestURL, "/")
	if len(parts) == 0 {
		return 0
	}
	return parseInt(parts[len(parts)-1])
}

func fileExists(path string) bool {
	return fs.IsFile(path).OK
}
