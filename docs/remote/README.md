<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Remote dispatch

Run a dispatch on **another** `core-agent` node over its HTTP MCP endpoint, then poll it
from here. The remote node executes the normal [dispatch](../dispatch/) →
[closeout](../pipeline/) flow; this side only initiates and watches.

| Tool | What it does |
|------|--------------|
| `agentic_dispatch_remote` | proxy a dispatch to a remote node (HTTP MCP) |
| `agentic_status_remote` | poll the remote dispatch's status |

Use it to send work to the node that owns the repo, has the GPU, or is the homelab box.
The target node must have its queue running — after a restart, `agentic_dispatch_start`
on that node unfreezes it.

This is part of the fleet story — see [fleet](../fleet/) for registration, `agents.yaml`,
and repo sync.
