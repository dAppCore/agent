<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Monitor — background monitoring & repo sync

`monitor` (`pkg/monitor/`) runs the background loops that keep the agent's world current.

- **Completion harvest** (`harvest.go`) — watches for dispatched-agent completion signals
  and feeds them into the [closeout pipeline](../pipeline/).
- **Monitor API** (`monitor.go`) — exposes monitoring state.
- **Repo sync** (`sync.go`) — keeps ecosystem repos fresh against `agents.yaml`:
  - `syncRepos()` — pull/refresh the repos this machine owns.
  - `syncWorkspacePush(repo, branch, org)` — push a workspace branch back.
  - `initSyncTimestamp()` — incremental syncs (only what changed since last time).

This is the engine behind the [fleet](../fleet/) repo-sync story and the reason a
finished dispatch flows into closeout without manual polling. System view:
[`../architecture.md`](../architecture.md).
