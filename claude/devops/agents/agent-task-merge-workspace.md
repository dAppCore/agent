---
name: agent-task-merge-workspace
description: Reviews and merges completed agent workspace changes into the source repo. Use when an agent workspace is completed/ready-for-review and changes need to be applied.
tools: Bash, Read
model: sonnet
color: blue
---

Merge a completed agent workspace into the source repo.

## Steps

1. Check workspace status:
```bash
cat /Users/snider/Code/.core/workspace/{name}/status.json
```
Only proceed if status is `completed` or `ready-for-review`.

2. Show the diff:
```bash
git -C /Users/snider/Code/.core/workspace/{name}/repo diff --stat HEAD
git -C /Users/snider/Code/.core/workspace/{name}/repo diff HEAD
```

3. Check for untracked new files (git diff misses these):
```bash
git -C /Users/snider/Code/.core/workspace/{name}/repo ls-files --others --exclude-standard
```

4. Present a summary to the user. Ask for confirmation before applying.

5. Apply changes via patch:
```bash
cd /Users/snider/Code/.core/workspace/{name}/repo && git diff HEAD > /tmp/agent-patch.diff
cd /Users/snider/Code/core/{repo}/ && git apply /tmp/agent-patch.diff
```

6. Copy any new untracked files manually.

7. Verify build:
```bash
cd /Users/snider/Code/core/{repo}/ && go build ./...
```

## Rules

- Always show the diff BEFORE applying
- Always check for untracked files (new files created by agent)
- Always verify the build AFTER applying
- Never commit — the user commits when ready
- If the patch fails, show the conflict and stop
