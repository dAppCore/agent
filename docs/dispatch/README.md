<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Dispatch

**Dispatch** is core/agent's core loop: it takes a tracked issue, preps an isolated
workspace, runs a coding agent inside it, and watches it to completion — which then
triggers the [closeout pipeline](../pipeline/). It's how work gets from a tracker into a
merged PR with no human in the loop.

## The flow

```
agentic_scan          find tracked issues
  → agentic_dispatch  prep an isolated workspace, resolve + run the runner
  → runner edits, commits, pushes
  → completion → closeout pipeline (QA → PR → verify → merge)
```

## Dispatching

```
agentic_dispatch(repo, task="<what to do>", agent="codex:gpt-5.4-mini",
                 branch="dev", template="coding")
```

The workspace lands at `.core/workspace/<org>/<repo>/task-<N>`; the call returns the
workspace dir, runner PID, and an output file. **Which runner runs, and whether it runs
on the host or in a container, is decided by the `agent` string — see
[runners](runners.md).**

## The dispatch queue

| Tool | What it does |
|------|--------------|
| `agentic_dispatch_start` | start the queue — **run after a restart to unfreeze it** |
| `agentic_dispatch_shutdown` / `_shutdown_now` | drain + stop / stop immediately |

## In this section

- [runners](runners.md) — native-vs-container, the `provider:model` string, runtimes.

**Related:** [pipeline](../pipeline/) (what runs at completion) · [scan-mirror](../scan-mirror/)
(`agentic_scan`) · [fleet](../fleet/) (remote dispatch) · [plans](../plans/) (multi-issue
orchestration).
