<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Fleet & remote dispatch

A **fleet** is several `core-agent` machines that share the PHP backend and can hand work
to each other — so a dispatch can run on the node that owns the repo or has the GPU. This
page covers joining the fleet and keeping repos in sync; remote dispatch has its own
[page](../remote/).

## Defined by `agents.yaml`

`agents.yaml` (`agentic.AgentsConfigPath()`) lists the machines and the repos each works;
`core-agent check` reports whether it's present.

## Registration

A machine joins via the **TLS-validating** shared client (`transport.go:defaultClient` —
cert validation on):

| Endpoint | Purpose |
|----------|---------|
| `POST /v1/fleet/register` | register this machine |
| `POST /v1/fleet/heartbeat` | liveness |

Inspect it: `agentic:fleet/nodes` (list machines) · `agentic:fleet/status` (health).

## Repo sync

The [monitor](../monitor/) subsystem keeps repos fresh against `agents.yaml`
(`syncRepos`, `syncWorkspacePush`, incremental via `initSyncTimestamp`). `agentic:repo/sync`
freshens one repo on demand before a dispatch.

## In this section

- [remote](../remote/) — proxying a dispatch to another node over HTTP MCP.

**Related:** [monitor](../monitor/) (the sync engine) · [dispatch](../dispatch/) ·
[plans](../plans/) (sessions resume across the shared backend).
