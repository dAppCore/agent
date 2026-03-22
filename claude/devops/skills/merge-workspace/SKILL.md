---
name: merge-workspace
description: This skill should be used when the user asks to "merge workspace", "bring over the changes", "review and merge agent work", "pull in the agent's diff", or needs to take a completed agent workspace and merge its changes into the dev branch. Checks the diff, verifies the build, and applies changes.
argument-hint: <workspace-name>
allowed-tools: ["Bash", "Read"]
---

# Merge Agent Workspace

Take a completed agent workspace and merge its changes into the repo's dev branch.

## Steps

1. Find the workspace. The argument is the workspace name (e.g., `agent-1774177216443021000`). The full path is:
   ```
   /Users/snider/Code/.core/workspace/<name>/
   ```

2. Check status:
   ```bash
   cat /Users/snider/Code/.core/workspace/<name>/status.json
   ```
   Only proceed if status is `completed` or `ready-for-review`.

3. Show the diff from the workspace's src/ directory:
   ```bash
   cd /Users/snider/Code/.core/workspace/<name>/src && git diff HEAD
   ```
   Present a summary of what changed to the user.

4. Ask the user if the changes look good before proceeding.

5. Find the original repo path from status.json (`repo` field). The repo lives at:
   ```
   /Users/snider/Code/core/<repo>/
   ```

6. Cherry-pick or apply the changes. Use `git diff` to create a patch and apply it:
   ```bash
   cd /Users/snider/Code/.core/workspace/<name>/src && git diff HEAD > /tmp/agent-patch.diff
   cd /Users/snider/Code/core/<repo>/ && git apply /tmp/agent-patch.diff
   ```

7. Verify the build passes:
   ```bash
   cd /Users/snider/Code/core/<repo>/ && go build ./...
   ```

8. Report results. Do NOT commit — let the user decide when to commit.

## Important

- Always show the diff BEFORE applying
- Always verify the build AFTER applying
- Never commit — the user commits when ready
- If the patch fails to apply, show the conflict and stop
