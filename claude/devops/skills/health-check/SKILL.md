---
name: health-check
description: This skill should be used when the user asks to "check health", "is core-agent working", "check agent", "health check", "system status", or needs to verify the core-agent binary, config, and workspace are healthy.
argument-hint: (no arguments needed)
allowed-tools: ["Bash"]
---

# Core Agent Health Check

Verify binary, agents.yaml, workspace directory, and environment.

```bash
core-agent check
```

Also useful:
```bash
# Show all environment keys and values
core-agent env

# Show version and build info
core-agent version
```
