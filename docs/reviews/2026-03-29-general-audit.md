<!-- SPDX-License-Identifier: EUPL-1.2 -->

# General Audit — 2026-03-29

## Scope

General review of code quality, architecture, and correctness in the Go orchestration path.

- Requested `CODEX.md` was not present anywhere under `/workspace`, so the review used `CLAUDE.md`, `AGENTS.md`, and the live code paths instead.
- Automated checks run from a clean worktree:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./... -count=1 -timeout 60s`

## Automated Check Result

All three Go commands fail immediately because the repo mixes the new `forge.lthn.ai/core/mcp` module requirement with old `dappco.re/go/mcp/...` imports. The failure reproduced from a clean checkout before any local edits.

## Findings

### 1. High — the repo does not currently build because the MCP dependency path is inconsistent

`go.mod:12` requires `forge.lthn.ai/core/mcp`, but the source still imports `dappco.re/go/mcp/...` in multiple packages such as `cmd/core-agent/main.go:10`, `pkg/brain/brain.go:12`, `pkg/brain/direct.go:11`, `pkg/monitor/monitor.go:21`, and `pkg/runner/runner.go:18`.

Impact:

- `go build ./...`, `go vet ./...`, and `go test ./...` all fail before package compilation starts.
- This blocks every other correctness check and makes the repo unreleasable in its current state.

Recommendation:

- Pick one canonical MCP module path and update both `go.mod` and imports together.
- Add a CI guard that runs `go list ./...` or `go build ./...` before merge so module-path drift cannot land again.

### 2. High — resuming an existing workspace forcibly checks out `main`, which abandons the agent branch and breaks non-`main` repos

`pkg/agentic/prep.go:433` to `pkg/agentic/prep.go:436` now does:

- `git checkout main`
- `git pull origin main`

This happens before the code reads the existing branch back out at `pkg/agentic/prep.go:470` to `pkg/agentic/prep.go:472`.

Impact:

- A resumed workspace that was previously on `agent/...` is silently moved back to `main`.
- The resumed agent can continue on the wrong branch, making its follow-up commit land on the base branch instead of the workspace branch.
- Repos whose default branch is `dev` or anything other than `main` will fail this resume path outright.

Recommendation:

- Preserve the existing branch and update it explicitly, or rebase/merge the default branch into the current workspace branch.
- Add a regression test for resuming an `agent/...` branch and for repos whose default branch is `dev`.

### 3. High — one agent completion can mark every running workspace for the same repo as completed

In `pkg/runner/runner.go:136` to `pkg/runner/runner.go:143`, the `AgentCompleted` handler updates the in-memory registry by `Repo` only:

- any `running` workspace whose `st.Repo == ev.Repo` is marked with the completed status
- `ev.Workspace` is ignored even though it is already included in the event payload

Impact:

- Two concurrent tasks against the same repo are not isolated.
- When one finishes, the other can be marked completed early, its PID is cleared, and concurrency accounting drops too soon.
- Queue drain and status reporting can then dispatch more work even though a task is still running.

Recommendation:

- Use the workspace identifier as the primary key when applying lifecycle events.
- Add a test with two running workspaces for the same repo and assert only the matching workspace changes state.

### 4. High — the monitor harvest pipeline still looks for `src/`, so real completed workspaces never transition to `ready-for-review`

Workspace prep clones the checkout into `repo/` at `pkg/agentic/prep.go:414` to `pkg/agentic/prep.go:415` and later uses that same directory throughout dispatch and resume. But `pkg/monitor/harvest.go:91` still reads the workspace from `wsDir + "/src"`.

The tests reinforce the old layout instead of the real one: `pkg/monitor/harvest_test.go:29` to `pkg/monitor/harvest_test.go:33` creates fixtures under `src/`.

Impact:

- `harvestWorkspace` returns early for real workspaces because `repo/` exists and `src/` does not.
- Completed agents never move to `ready-for-review`, so the monitor's review handoff is effectively dead.
- The current tests give false confidence because they only exercise the obsolete directory layout.

Recommendation:

- Switch harvest to `repo/` or a shared path helper used by both prep and monitor.
- Rewrite the monitor fixtures to match actual workspaces produced by `prepWorkspace`.

### 5. Medium — status and resume still assume the old flat log location, so dead agents are misclassified and resume returns the wrong log path

Actual agent logs are written under `.meta` by `pkg/agentic/dispatch.go:213` to `pkg/agentic/dispatch.go:215`, but:

- `pkg/agentic/status.go:155` reads `wsDir/agent-<agent>.log`
- `pkg/agentic/resume.go:114` returns that same old path in `ResumeOutput`

Impact:

- If a process exits and `BLOCKED.md` is absent, `agentic_status` can mark the workspace `failed` even though `.meta/agent-*.log` exists and should imply normal completion.
- Callers that trust `ResumeOutput.OutputFile` are pointed at a file that is never written.

Recommendation:

- Replace these call sites with the shared `agentOutputFile` helper.
- Add a status test that writes only `.meta/agent-codex.log` and verifies the workspace becomes `completed`, not `failed`.

### 6. Medium — workspace discovery is still shallow in watch and CLI code, and the action wrapper drops the explicit workspace argument entirely

The newer nested layout is `workspace/{org}/{repo}/{task}`. Several user-facing entry points still only scan `workspace/*/status.json` or use `PathBase`:

- `pkg/agentic/watch.go:194` to `pkg/agentic/watch.go:204`
- `pkg/agentic/commands_workspace.go:25` and `pkg/agentic/commands_workspace.go:52`

Separately, `pkg/agentic/actions.go:113` to `pkg/agentic/actions.go:115` constructs `WatchInput{}` and ignores the caller's `workspace` option completely.

Impact:

- `agentic_watch` without explicit workspaces can miss active nested workspaces.
- `workspace/list` and `workspace/clean` miss or mis-handle most real workspaces under the new layout.
- `core-agent` action callers cannot actually watch a specific workspace even though the action comment says they can.

Recommendation:

- Use the same shallow+deep glob strategy already used in `status`, `prep`, and `runner`.
- Thread the requested workspace through `handleWatch` and normalise on relative workspace paths rather than `PathBase`.

## Architectural Note

Several of the defects above come from the same root cause: the codebase has partially migrated from older workspace conventions (`src/`, flat workspace names, flat log files) to newer ones (`repo/`, nested `org/repo/task` paths, `.meta` logs), but the path logic is duplicated across services instead of centralised.

The highest-leverage clean-up would be a single shared workspace-path helper layer used by:

- prep and resume
- runner and monitor
- status, watch, and CLI commands
- log-file lookup and event key generation

That would remove the current class of half-migrated path regressions.
