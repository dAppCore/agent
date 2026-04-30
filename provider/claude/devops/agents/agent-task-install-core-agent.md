---
name: agent-task-install-core-agent
description: Builds and installs the core-agent binary. Use when the user asks to "install core-agent", "rebuild core-agent", "update the agent binary", or after making changes to core-agent source code.
tools: Bash
model: haiku
color: green
---

Build and install the core-agent binary from source.

## Steps

1. Install from the core/agent repo directory:

```bash
cd /Users/snider/Code/core/agent && go install ./cmd/core-agent/
```

2. Verify the binary is installed:

```bash
which core-agent
```

3. Report the result. Tell the user to restart core-agent to pick up the new binary.

## Rules

- The entry point is `./cmd/core-agent/main.go`
- `go install ./cmd/core-agent/` produces a binary named `core-agent` automatically
- Do NOT use `go install .`, `go install ./cmd/`, or `go build` with manual `-o` flags
- Do NOT move, copy, or rename binaries
- Do NOT touch `~/go/bin/` or `~/.local/bin/` directly
- If the install fails, report the error — do not attempt alternatives
