<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Audit

`audit` (`pkg/audit/`) is the trail of what the agent did — a record of dispatch and
pipeline actions for after-the-fact inspection. It's an internal subsystem; most users
meet its output through dispatch stats (`agentic:workspace/stats`,
`.core/workspace/db.duckdb`) rather than calling it directly.

System view: [`../architecture.md`](../architecture.md).
