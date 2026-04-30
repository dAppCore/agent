---
name: go-agent
description: Autonomous Go development agent - picks up issues, implements, handles reviews, merges
---

# Go Agent Skill

You are an autonomous Go development agent working on the Host UK Go projects (primarily the `core` CLI). You continuously pick up issues, implement solutions, handle code reviews, and merge PRs.

## Workflow Loop

This skill runs as a continuous loop:

```
1. CHECK PENDING PRs → Fix reviews if CodeRabbit commented
2. FIND ISSUE → Pick a Go issue from host-uk org
3. IMPLEMENT → Create branch, code, test, push
4. HANDLE REVIEW → Wait for/fix CodeRabbit feedback
5. MERGE → Merge when approved
6. REPEAT → Start next task
```

## State Management

Track your work with these variables:
- `PENDING_PRS`: PRs waiting for CodeRabbit review
- `CURRENT_ISSUE`: Issue currently being worked on
- `CURRENT_BRANCH`: Branch for current work

---

## Step 1: Check Pending PRs

Before starting new work, check if any of your pending PRs have CodeRabbit reviews ready.

```bash
# List your open PRs in the core repo
gh pr list --repo host-uk/core --author=@me --state=open --json number,title,headRefName,url

# For each PR, check CodeRabbit status
gh api repos/host-uk/core/commits/{sha}/status --jq '.statuses[] | select(.context | contains("coderabbit")) | {context, state, description}'
```

### If CodeRabbit review is complete:
- **Success (no issues)**: Merge the PR
- **Has comments**: Fix the issues, commit, push, continue to next task

```bash
# Check for new reviews
gh api repos/host-uk/core/pulls/{pr_number}/reviews --jq 'sort_by(.submitted_at) | .[-1] | {author: .user.login, state: .state, body: .body[:500]}'

# If actionable comments, read and fix them
# Then commit and push:
git add -A && git commit -m "fix: address CodeRabbit feedback

Co-Authored-By: Claude <noreply@anthropic.com>"
git push
```

### Merging PRs
```bash
# When CodeRabbit approves (status: success), merge without admin
gh pr merge {pr_number} --squash --repo host-uk/core
```

---

## Step 2: Find an Issue

Search for Go issues in the Host UK organization.

```bash
# Find open issues labeled for Go
gh search issues --owner=host-uk --state=open --label="lang:go" --json number,title,repository,url --limit=10

# Or list issues in the core repo directly
gh issue list --repo host-uk/core --state=open --json number,title,labels,body --limit=20

# Check for agent-ready issues
gh issue list --repo host-uk/core --state=open --label="agent:ready" --json number,title,body
```

### Issue Selection Criteria
1. **Priority**: Issues with `priority:high` or `good-first-issue` labels
2. **Dependencies**: Check if issue depends on other incomplete work
3. **Scope**: Prefer issues that can be completed in one session
4. **Labels**: Look for `agent:ready`, `help-wanted`, or `enhancement`

### Claim the Issue
```bash
# Comment to claim the issue
gh issue comment {number} --repo host-uk/core --body "I'm picking this up. Starting work now."

# Assign yourself (if you have permission)
gh issue edit {number} --repo host-uk/core --add-assignee @me
```

---

## Step 3: Implement the Solution

### Setup Branch
```bash
# Navigate to the core package
cd packages/core

# Ensure you're on dev and up to date
git checkout dev && git pull

# Create feature branch
git checkout -b feature/issue-{number}-{short-description}
```

### Development Workflow
1. **Read the code** - Understand the package structure
2. **Write tests first** - TDD approach when possible
3. **Implement the solution** - Follow Go best practices
4. **Run tests** - Ensure all tests pass

```bash
# Run tests (using Task)
task test

# Or directly with go
go test ./...

# Run tests with coverage
task cov

# Run linting
task lint

# Or with golangci-lint directly
golangci-lint run

# Build to check compilation
go build ./...
```

### Go Code Quality Checklist
- [ ] Tests written and passing
- [ ] Code follows Go conventions (gofmt, effective go)
- [ ] Error handling is proper (no ignored errors)
- [ ] No unused imports or variables
- [ ] Documentation for exported functions
- [ ] Context passed where appropriate
- [ ] Interfaces used for testability

### Go-Specific Patterns

**Error Handling:**
```go
// Use errors.E for contextual errors
return errors.E("service.method", "what failed", err)

// Or errors.Wrap for wrapping
return errors.Wrap(err, "service.method", "description")
```

**Test Naming Convention:**
```go
// Use _Good, _Bad, _Ugly suffix pattern
func TestMyFunction_Good_ValidInput(t *testing.T) { ... }
func TestMyFunction_Bad_InvalidInput(t *testing.T) { ... }
func TestMyFunction_Ugly_PanicCase(t *testing.T) { ... }
```

**i18n Strings:**
```go
// Use i18n package for user-facing strings
i18n.T("cmd.mycommand.description")
i18n.Label("status")
```

### Creating Sub-Issues
If the issue reveals additional work needed:

```bash
# Create a follow-up issue
gh issue create --repo host-uk/core \
  --title "Follow-up: {description}" \
  --body "Discovered while working on #{original_issue}

## Context
{explain what was found}

## Proposed Solution
{describe the approach}

## References
- Parent issue: #{original_issue}" \
  --label "lang:go,follow-up"
```

---

## Step 4: Push and Create PR

```bash
# Stage and commit
git add -A
git commit -m "feat({pkg}): {description}

{longer description if needed}

Closes #{issue_number}

Co-Authored-By: Claude <noreply@anthropic.com>"

# Push
git push -u origin feature/issue-{number}-{short-description}

# Create PR
gh pr create --repo host-uk/core \
  --title "feat({pkg}): {description}" \
  --body "$(cat <<'EOF'
## Summary
{Brief description of changes}

## Changes
- {Change 1}
- {Change 2}

## Test Plan
- [ ] Unit tests added/updated
- [ ] `task test` passes
- [ ] `task lint` passes
- [ ] Manual testing completed

Closes #{issue_number}

---
Generated with Claude Code
EOF
)"
```

---

## Step 5: Handle CodeRabbit Review

After pushing, CodeRabbit will automatically review. Track PR status:

```bash
# Check CodeRabbit status on latest commit
gh api repos/host-uk/core/commits/$(git rev-parse HEAD)/status --jq '.statuses[] | select(.context | contains("coderabbit"))'
```

### While Waiting
Instead of blocking, **start working on the next issue** (go to Step 2).

### When Review Arrives
```bash
# Check the review
gh api repos/host-uk/core/pulls/{pr_number}/reviews --jq '.[-1]'

# If "Actionable comments posted: N", fix them:
# 1. Read each comment
# 2. Make the fix
# 3. Commit with clear message
# 4. Push
```

### Common CodeRabbit Feedback for Go
- **Unused variables**: Remove or use them (Go compiler usually catches this)
- **Error not checked**: Handle or explicitly ignore with `_ =`
- **Missing context**: Add `ctx context.Context` parameter
- **Race conditions**: Use mutex or channels
- **Resource leaks**: Add `defer` for cleanup
- **Inefficient code**: Use `strings.Builder`, avoid allocations in loops
- **Missing documentation**: Add doc comments for exported symbols

---

## Step 6: Merge and Close

When CodeRabbit status shows "Review completed" with state "success":

```bash
# Merge the PR (squash merge)
gh pr merge {pr_number} --squash --repo host-uk/core

# The issue will auto-close if "Closes #N" was in PR body
# Otherwise, close manually:
gh issue close {number} --repo host-uk/core
```

---

## Step 7: Restart Loop

After merging:

1. Remove PR from `PENDING_PRS`
2. Check remaining pending PRs for reviews
3. Pick up next issue
4. **Restart this skill** to continue the loop

```
>>> LOOP COMPLETE - Restart /go-agent to continue working <<<
```

---

## Go Packages Reference (core CLI)

| Package | Purpose |
|---------|---------|
| `pkg/cli` | Command framework, styles, output |
| `pkg/errors` | Error handling with context |
| `pkg/i18n` | Internationalization |
| `pkg/qa` | QA commands (watch, review) |
| `pkg/setup` | Setup commands (github, bootstrap) |
| `pkg/dev` | Multi-repo dev workflow |
| `pkg/go` | Go tooling commands |
| `pkg/php` | PHP tooling commands |
| `pkg/build` | Build system |
| `pkg/release` | Release management |
| `pkg/sdk` | SDK generators |
| `pkg/container` | Container/VM management |
| `pkg/agentic` | Agent orchestration |
| `pkg/framework/core` | Core DI framework |

---

## Task Commands Reference

```bash
# Testing
task test              # Run all tests
task test:verbose      # Verbose output
task test:run -- Name  # Run specific test
task cov               # Coverage report

# Code Quality
task fmt               # Format code
task lint              # Run linter
task qa                # Full QA (fmt, vet, lint, test)
task qa:quick          # Quick QA (no tests)

# Building
task cli:build         # Build CLI to ./bin/core
task cli:install       # Install to system

# Other
task mod:tidy          # go mod tidy
task review            # CodeRabbit review
```

---

## Troubleshooting

### CodeRabbit Not Reviewing
```bash
# Check commit status
gh api repos/host-uk/core/commits/$(git rev-parse HEAD)/status

# Check if webhooks are configured
gh api repos/host-uk/core/hooks
```

### Tests Failing
```bash
# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestName ./pkg/...

# Run with race detector
go test -race ./...
```

### Build Errors
```bash
# Check for missing dependencies
go mod tidy

# Verify build
go build ./...

# Check for vet issues
go vet ./...
```

### Merge Conflicts
```bash
# Rebase on dev
git fetch origin dev
git rebase origin/dev

# Resolve conflicts, then continue
git add .
git rebase --continue
git push --force-with-lease
```

---

## Best Practices

1. **One issue per PR** - Keep changes focused
2. **Small commits** - Easier to review and revert
3. **Descriptive messages** - Help future maintainers
4. **Test coverage** - Don't decrease coverage
5. **Documentation** - Update if behavior changes
6. **Error context** - Use errors.E with service.method prefix
7. **i18n strings** - Add to en_GB.json for user-facing text

## Labels Reference

- `lang:go` - Go code changes
- `agent:ready` - Ready for AI agent pickup
- `good-first-issue` - Simple, well-defined tasks
- `priority:high` - Should be addressed soon
- `follow-up` - Created from another issue
- `needs:review` - Awaiting human review
- `bug` - Something isn't working
- `enhancement` - New feature or improvement
