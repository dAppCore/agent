---
name: install-core-agent
description: This skill should be used when the user asks to "install core-agent", "rebuild core-agent", "update the agent binary", or needs to compile and install the core-agent MCP server binary. Runs the correct go install from the right directory with the right path.
argument-hint: (no arguments needed)
allowed-tools: ["Bash"]
---

# Install core-agent

Build and install the core-agent binary from source.

## Steps

1. Run from the core/agent repo directory:

```bash
cd /Users/snider/Code/core/agent && go install ./cmd/core-agent/
```

2. Verify the binary is installed:

```bash
which core-agent
```

3. Tell the user to restart core-agent (it runs as an MCP server — the process needs restarting to pick up the new binary).

## Important

- The entry point is `./cmd/core-agent/main.go` — NOT `./cmd/` or `.`
- `go install ./cmd/core-agent/` produces a binary named `core-agent` automatically
- Do NOT use `go install .`, `go install ./cmd/`, or `go build` with manual `-o` flags
- Do NOT move, copy, or rename binaries
- Do NOT touch `~/go/bin/` or `~/.local/bin/` directly
- If the install fails, report the error to the user — do not attempt alternatives
