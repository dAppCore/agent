// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sync"
	"testing"
	"time"

	core "dappco.re/go"
)

func TestFetchLoop_RunFetchLoop_Good_TicksAtConfiguredInterval(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 25ms\n",
		"repos:\n",
		"  - good-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	interval := subsystem.fetchLoopInterval()
	core.AssertEqual(t, 25*time.Millisecond, interval)
	state := stubFetchLoop(t, "")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, interval)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	core.AssertEqual(t, 0, state.count("good-repo"))

	waitForFetchLoopCalls(t, state, "good-repo", 2, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)
}

func TestFetchLoop_RunFetchLoop_Bad_SurvivesFailingFetch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	fetchLoopCreateRepo(t, codePath, "core", "bad-repo")
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 15ms\n",
		"repos:\n",
		"  - good-repo\n",
		"  - bad-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	state := stubFetchLoop(t, "bad-repo")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, subsystem.fetchLoopInterval())
		close(done)
	}()

	waitForFetchLoopCalls(t, state, "bad-repo", 1, 250*time.Millisecond)
	waitForFetchLoopCalls(t, state, "good-repo", 2, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)
}

func TestFetchLoop_RunFetchLoop_Ugly_StopsOnContextCancel(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	core.RequireTrue(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 15ms\n",
		"repos:\n",
		"  - good-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	state := stubFetchLoop(t, "")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, subsystem.fetchLoopInterval())
		close(done)
	}()

	waitForFetchLoopCalls(t, state, "good-repo", 1, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)

	countAfterCancel := state.count("good-repo")
	time.Sleep(50 * time.Millisecond)
	core.AssertEqual(t, countAfterCancel, state.count("good-repo"))
}

func fetchLoopTestPrep(codePath string) *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

type fetchLoopCallState struct {
	mu    sync.Mutex
	calls map[string]int
	fail  string
}

func stubFetchLoop(t *testing.T, failRepo string) *fetchLoopCallState {
	t.Helper()

	state := &fetchLoopCallState{
		calls: map[string]int{},
		fail:  failRepo,
	}
	previousBranch := fetchLoopDefaultBranchFunc
	previousFetch := fetchLoopRunFetchFunc

	fetchLoopDefaultBranchFunc = func(_ *PrepSubsystem, _ string) string {
		return "dev"
	}
	fetchLoopRunFetchFunc = func(_ *PrepSubsystem, ctx context.Context, repoDir, _ string) core.Result {
		repo := core.PathBase(repoDir)
		state.mu.Lock()
		state.calls[repo]++
		state.mu.Unlock()
		if err := ctx.Err(); err != nil {
			return core.Fail(core.E("fetchLoopRunFetch", "context canceled", err))
		}
		if repo == state.fail {
			return core.Fail(core.E("fetchLoopRunFetch", "fetch failed", nil))
		}
		return core.Ok(nil)
	}

	t.Cleanup(func() {
		fetchLoopDefaultBranchFunc = previousBranch
		fetchLoopRunFetchFunc = previousFetch
	})

	return state
}

func (s *fetchLoopCallState) count(repo string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls[repo]
}

func waitForFetchLoopCalls(t *testing.T, state *fetchLoopCallState, repo string, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if state.count(repo) >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	core.AssertGreaterOrEqual(t, state.count(repo), want)
}

func fetchLoopWriteGitScript(t *testing.T, logPath, badRepo string) {
	t.Helper()

	binDir := t.TempDir()
	gitPath := core.JoinPath(binDir, "git")
	script := core.Concat(
		"#!/bin/sh\n",
		"repo=$(basename \"$(pwd)\")\n",
		"printf '%s|%s\\n' \"$repo\" \"$*\" >> ", logPath, "\n",
		"if [ \"$1\" = \"symbolic-ref\" ]; then\n",
		"  printf 'origin/dev\\n'\n",
		"  exit 0\n",
		"fi\n",
		"if [ \"$1\" = \"fetch\" ] && [ \"$repo\" = \"", badRepo, "\" ]; then\n",
		"  exit 1\n",
		"fi\n",
		"exit 0\n",
	)
	core.RequireTrue(t, fs.Write(gitPath, script).OK)
	core.RequireTrue(t, testCore.Process().RunIn(context.Background(), binDir, "chmod", "+x", gitPath).OK)
	t.Setenv("PATH", core.Concat(binDir, ":", core.Env("PATH")))
}

func fetchLoopCreateRepo(t *testing.T, codePath, org, repo string) {
	t.Helper()
	repoDir := core.JoinPath(codePath, org, repo)
	core.RequireTrue(t, fs.EnsureDir(core.JoinPath(repoDir, ".git")).OK)
}

func fetchLoopWaitForCount(t *testing.T, logPath, repo, snippet string, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fetchLoopLogCount(logPath, repo, snippet) >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	core.AssertGreaterOrEqual(t, fetchLoopLogCount(logPath, repo, snippet), want)
}

func fetchLoopWaitForDone(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("fetch loop did not stop after cancellation")
	}
}

func fetchLoopLogCount(logPath, repo, snippet string) int {
	readResult := fs.Read(logPath)
	if !readResult.OK {
		return 0
	}

	content := core.Trim(readResult.Value.(string))
	if content == "" {
		return 0
	}

	count := 0
	for _, line := range core.Split(content, "\n") {
		if repo != "" && !core.HasPrefix(line, core.Concat(repo, "|")) {
			continue
		}
		if snippet != "" && !core.Contains(line, snippet) {
			continue
		}
		count++
	}

	return count
}
