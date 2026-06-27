<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Fleet & remote dispatch — many machines, one backend

A "fleet" is several `core-agent` machines that share the PHP backend and can hand work
to each other. This guide covers registering a machine, keeping its repos in sync, and
proxying a dispatch to another node.

## The fleet is defined by `agents.yaml`

`agents.yaml` (`agentic.AgentsConfigPath()`) lists the machines and the repos each works.
`core-agent check` reports whether it's present.

## Registration + heartbeat

A machine joins by posting to the backend through the **TLS-validating shared client**
(`transport.go:defaultClient` — certificate validation is on, not skipped):

| Endpoint | Purpose |
|----------|---------|
| `POST /v1/fleet/register` | register this machine into the fleet |
| `POST /v1/fleet/heartbeat` | keep-alive / liveness |

Inspect the fleet:

```
agentic:fleet/nodes     # list the registered machines
agentic:fleet/status    # fleet health/status
```

(Both have bare `fleet/nodes` / `fleet/status` aliases too.)

## Repo sync

The [monitor](../monitor/) subsystem keeps the ecosystem repos fresh against
`agents.yaml`:

- `Subsystem.syncRepos()` — pull/refresh the repos this machine is responsible for.
- `Subsystem.syncWorkspacePush(repo, branch, org)` — push a workspace branch back.
- `initSyncTimestamp()` — tracks last-sync so syncs are incremental.

`agentic:repo/sync` freshens a single repo on demand (used before a dispatch so the
workspace starts from a clean, current tree).

## Remote dispatch

A dispatch can be proxied to **another** `core-agent` over its HTTP MCP endpoint — the
node that owns the repo (or has the GPU) does the work:

| Tool | What it does |
|------|--------------|
| `agentic_dispatch_remote` | run a dispatch on a remote node over HTTP MCP |
| `agentic_status_remote` | poll the remote dispatch's status |

The remote node runs the normal [dispatch](../dispatch/) → [closeout](../pipeline/) flow;
this side just polls. Remember the queue lifecycle: after a node restarts, run
`agentic_dispatch_start` there to unfreeze its queue (see [dispatch](../dispatch/)).

## Next

[dispatch](../dispatch/) · [monitor](../monitor/) (the sync engine) ·
[plans](../plans/) (sessions resume across the fleet because state is backend-held).
