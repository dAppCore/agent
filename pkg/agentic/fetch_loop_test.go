// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchLoop_RunFetchLoop_Good_TicksAtConfiguredInterval(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	logPath := core.JoinPath(t.TempDir(), "git.log")
	fetchLoopWriteGitScript(t, logPath, "bad-repo")
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 25ms\n",
		"repos:\n",
		"  - good-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	interval := subsystem.fetchLoopInterval()
	assert.Equal(t, 25*time.Millisecond, interval)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, interval)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 0, fetchLoopLogCount(logPath, "good-repo", "fetch origin dev"))

	fetchLoopWaitForCount(t, logPath, "good-repo", "fetch origin dev", 2, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)
}

func TestFetchLoop_RunFetchLoop_Bad_SurvivesFailingFetch(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	logPath := core.JoinPath(t.TempDir(), "git.log")
	fetchLoopWriteGitScript(t, logPath, "bad-repo")
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	fetchLoopCreateRepo(t, codePath, "core", "bad-repo")
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 15ms\n",
		"repos:\n",
		"  - good-repo\n",
		"  - bad-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, subsystem.fetchLoopInterval())
		close(done)
	}()

	fetchLoopWaitForCount(t, logPath, "bad-repo", "fetch origin dev", 1, 250*time.Millisecond)
	fetchLoopWaitForCount(t, logPath, "good-repo", "fetch origin dev", 2, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)
}

func TestFetchLoop_RunFetchLoop_Ugly_StopsOnContextCancel(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	codePath := t.TempDir()
	logPath := core.JoinPath(t.TempDir(), "git.log")
	fetchLoopWriteGitScript(t, logPath, "bad-repo")
	fetchLoopCreateRepo(t, codePath, "core", "good-repo")
	require.True(t, fs.Write(core.JoinPath(root, "agents.yaml"), core.Concat(
		"version: 1\n",
		"dispatch:\n",
		"  fetch_interval: 15ms\n",
		"repos:\n",
		"  - good-repo\n",
	)).OK)

	subsystem := fetchLoopTestPrep(codePath)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		subsystem.runFetchLoop(ctx, subsystem.fetchLoopInterval())
		close(done)
	}()

	fetchLoopWaitForCount(t, logPath, "good-repo", "fetch origin dev", 1, 250*time.Millisecond)

	cancel()
	fetchLoopWaitForDone(t, done)

	countAfterCancel := fetchLoopLogCount(logPath, "good-repo", "fetch origin dev")
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, countAfterCancel, fetchLoopLogCount(logPath, "good-repo", "fetch origin dev"))
}

func fetchLoopTestPrep(codePath string) *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		codePath:       codePath,
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

func fetchLoopWriteGitScript(t *testing.T, logPath, badRepo string) {
	t.Helper()

	binDir := t.TempDir()
	gitPath := core.JoinPath(binDir, "git")
	script := core.Concat(
		"#!/bin/sh\n",
		"repo=$(basename \"$PWD\")\n",
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
	require.True(t, fs.Write(gitPath, script).OK)
	require.True(t, testCore.Process().RunIn(context.Background(), binDir, "chmod", "+x", gitPath).OK)
	t.Setenv("PATH", core.Concat(binDir, ":", core.Env("PATH")))
}

func fetchLoopCreateRepo(t *testing.T, codePath, org, repo string) {
	t.Helper()
	repoDir := core.JoinPath(codePath, org, repo)
	require.True(t, fs.EnsureDir(core.JoinPath(repoDir, ".git")).OK)
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

	require.GreaterOrEqual(t, fetchLoopLogCount(logPath, repo, snippet), want)
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
