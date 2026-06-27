<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Dispatch — fan an issue out to a sandboxed agent

Dispatch is the core loop: take a tracked issue (or a direct request), prep an isolated
workspace, run a coding agent in it, and watch it to completion. Completion then triggers
the [closeout pipeline](../pipeline/).

## The flow

```
agentic_scan            find tracked issues to work
  → agentic_dispatch    prep an isolated workspace, resolve + spawn the runner
  → runner edits, commits, pushes
  → completion detected → closeout pipeline (QA → PR → verify → merge)
```

## `agentic_dispatch`

The main tool/verb. Fans one issue out to a runner. Typical call:

```
agentic_dispatch(repo, task="<what to do>", agent="codex:gpt-5.4-mini",
                 branch="dev", template="coding")
```

- **`agent` is `provider[:model]`.** The provider picks the runner; the optional model
  after the colon is passed through — `codex:gpt-5.4-mini`, `claude:opus`,
  `opencode:gemma4-mlx-agentic`. Bare `codex` uses the provider default.
- Dispatch preps an **isolated workspace** under `.core/workspace/<org>/<repo>/task-<N>`
  and returns the workspace dir, the runner PID, and an output file. The
  `PrepSubsystem` tracks live workspaces (`OnStartup`/`OnShutdown`/`TrackWorkspace`).

### Native (host) vs containerised runners

| Runner | Where it runs |
|--------|---------------|
| `claude`, `coderabbit`, `opencode` | **on the host** (native) |
| `codex`, `gemini` | **inside a container** |

Container runtime is resolved by `containerCommandFor` across **Docker, Apple (VZ), and
Podman**, using the `core-dev` image and an optional GPU flag. An **unknown or empty
runtime name falls back to `docker`** so a dispatch never silently breaks. The
containerised agent runs `exec` in the workspace, with the model passed as `--model`.

## The dispatch queue

| Tool | What it does |
|------|--------------|
| `agentic_dispatch_start` | start the dispatch queue — **run this after a restart to unfreeze the queue** |
| `agentic_dispatch_shutdown` | drain + stop the queue gracefully |
| `agentic_dispatch_shutdown_now` | stop immediately |

## Scanning + remote

- `agentic_scan` — surface tracked (Forge) issues to dispatch against. See
  [scan-mirror](../scan-mirror/).
- `agentic_dispatch_remote` + `agentic_status_remote` — proxy a dispatch to another
  `core-agent` over HTTP MCP (the fleet path). See [fleet](../fleet/).

## CLI equivalents

Everything here has an `agentic:` CLI verb (and a bare alias): e.g. `agentic:issue/list`
to find work, `agentic:repo/sync` to freshen a workspace, `agentic:workspace/stats` for
the permanent dispatch stats in `.core/workspace/db.duckdb`.

## Next

When the runner finishes, control passes to the [closeout pipeline](../pipeline/).
For multi-issue / multi-agent orchestration see [plans](../plans/); for cross-machine
dispatch see [fleet](../fleet/).
