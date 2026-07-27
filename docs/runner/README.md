<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Runner — executing a dispatched agent

`runner` (`pkg/runner/`) is the internal subsystem that actually executes a dispatched
agent and tracks its workspace. Most users meet it only through [dispatch](../dispatch/);
this is what it does under the hood.

- Holds a `core.Registry[*WorkspaceStatus]` of live workspaces, plus a **dispatch lock**,
  a **drain lock**, and per-agent **backoff / fail counters** so a flapping agent backs
  off instead of hammering.
- Uses `c.Lock(name)` for named mutexes when the Core container is present, falling back
  to channel locks for standalone use.
- `queue.go` drains pending work; `paths.go` centralises workspace path resolution
  (`.core/workspace/<org>/<repo>/task-<N>`).

For the runtime decision (native-on-host vs containerised) see [dispatch](../dispatch/);
for the system view see [`../architecture.md`](../architecture.md).
