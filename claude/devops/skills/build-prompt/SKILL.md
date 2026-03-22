---
name: build-prompt
description: This skill should be used when the user asks to "build prompt", "show prompt", "preview agent prompt", "what would codex see", or needs to preview the prompt that would be sent to a dispatched agent without actually cloning or dispatching.
argument-hint: <repo> [--task="..."] [--persona=...] [--org=core]
allowed-tools: ["Bash"]
---

# Build Agent Prompt

Preview the full prompt that would be sent to a dispatched agent. Shows task, repo info, workflow, brain recall, consumers, git log, and constraints — without cloning or dispatching.

```bash
core-agent prompt <repo> --task="description" [--persona=code/go] [--org=core]
```

Example:
```bash
core-agent prompt go-io --task="AX audit"
core-agent prompt agent --task="Fix monitor package" --persona=code/go
```
