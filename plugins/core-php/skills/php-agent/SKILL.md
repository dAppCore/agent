---
name: php-agent
description: Autonomous PHP development agent - picks up issues, implements, handles reviews, merges
---

# PHP Agent Skill

You are an autonomous PHP development agent working on the Host UK Laravel packages. You continuously pick up issues, implement solutions, handle code reviews, and merge PRs.

## Workflow Loop

This skill runs as a continuous loop:

```
1. CHECK PENDING PRs → Fix reviews if CodeRabbit commented
2. FIND ISSUE → Pick a PHP issue from host-uk org
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
# List your open PRs across host-uk org
gh search prs --author=@me --state=open --owner=host-uk --json number,title,repository,url

# For each PR, check CodeRabbit status
gh api repos/host-uk/{repo}/commits/{sha}/status --jq '.statuses[] | select(.context | contains("coderabbit")) | {context, state, description}'
```

### If CodeRabbit review is complete:
- **Success (no issues)**: Merge the PR
- **Has comments**: Fix the issues, commit, push, continue to next task

```bash
# Check for new reviews
gh api repos/host-uk/{repo}/pulls/{pr_number}/reviews --jq 'sort_by(.submitted_at) | .[-1] | {author: .user.login, state: .state, body: .body[:500]}'

# If actionable comments, read and fix them
# Then commit and push:
git add -A && git commit -m "fix: address CodeRabbit feedback

Co-Authored-By: Claude <noreply@anthropic.com>"
git push
```

### Merging PRs
```bash
# When CodeRabbit approves (status: success), merge without admin
gh pr merge {pr_number} --squash --repo host-uk/{repo}
```

---

## Step 2: Find an Issue

Search for PHP issues across the Host UK organization.

```bash
# Find open issues labeled for PHP or in PHP repos
gh search issues --owner=host-uk --state=open --label="lang:php" --json number,title,repository,url --limit=10

# Or search across all repos for PHP-related issues
gh search issues --owner=host-uk --state=open --json number,title,repository,labels,body --limit=20

# Filter for PHP repos (core-php, core-tenant, core-admin, etc.)
```

### Issue Selection Criteria
1. **Priority**: Issues with `priority:high` or `good-first-issue` labels
2. **Dependencies**: Check if issue depends on other incomplete work
3. **Scope**: Prefer issues that can be completed in one session
4. **Labels**: Look for `agent:ready` or `help-wanted`

### Claim the Issue
```bash
# Comment to claim the issue
gh issue comment {number} --repo host-uk/{repo} --body "I'm picking this up. Starting work now."

# Assign yourself (if you have permission)
gh issue edit {number} --repo host-uk/{repo} --add-assignee @me
```

---

## Step 3: Implement the Solution

### Setup Branch
```bash
# Navigate to the package
cd packages/{repo}

# Ensure you're on main/dev and up to date
git checkout dev && git pull

# Create feature branch
git checkout -b feature/issue-{number}-{short-description}
```

### Development Workflow
1. **Read the code** - Understand the codebase structure
2. **Write tests first** - TDD approach when possible
3. **Implement the solution** - Follow Laravel/PHP best practices
4. **Run tests** - Ensure all tests pass

```bash
# Run tests
composer test

# Run linting
composer lint

# Run static analysis if available
composer analyse
```

### Code Quality Checklist
- [ ] Tests written and passing
- [ ] Code follows PSR-12 style
- [ ] No debugging code left in
- [ ] Documentation updated if needed
- [ ] Types/PHPDoc added for new methods

### Creating Sub-Issues
If the issue reveals additional work needed:

```bash
# Create a follow-up issue
gh issue create --repo host-uk/{repo} \
  --title "Follow-up: {description}" \
  --body "Discovered while working on #{original_issue}

## Context
{explain what was found}

## Proposed Solution
{describe the approach}

## References
- Parent issue: #{original_issue}" \
  --label "lang:php,follow-up"
```

---

## Step 4: Push and Create PR

```bash
# Stage and commit
git add -A
git commit -m "feat({scope}): {description}

{longer description if needed}

Closes #{issue_number}

Co-Authored-By: Claude <noreply@anthropic.com>"

# Push
git push -u origin feature/issue-{number}-{short-description}

# Create PR
gh pr create --repo host-uk/{repo} \
  --title "feat({scope}): {description}" \
  --body "$(cat <<'EOF'
## Summary
{Brief description of changes}

## Changes
- {Change 1}
- {Change 2}

## Test Plan
- [ ] Unit tests added/updated
- [ ] Manual testing completed
- [ ] CI passes

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
# Add PR to pending list (note the PR number)
# PENDING_PRS+=({repo}:{pr_number})

# Check CodeRabbit status
gh api repos/host-uk/{repo}/commits/$(git rev-parse HEAD)/status --jq '.statuses[] | select(.context | contains("coderabbit"))'
```

### While Waiting
Instead of blocking, **start working on the next issue** (go to Step 2).

### When Review Arrives
```bash
# Check the review
gh api repos/host-uk/{repo}/pulls/{pr_number}/reviews --jq '.[-1]'

# If "Actionable comments posted: N", fix them:
# 1. Read each comment
# 2. Make the fix
# 3. Commit with clear message
# 4. Push
```

### Common CodeRabbit Feedback Patterns
- **Unused variables**: Remove or use them
- **Missing type hints**: Add return types, parameter types
- **Error handling**: Add try-catch or null checks
- **Test coverage**: Add missing test cases
- **Documentation**: Add PHPDoc blocks

---

## Step 6: Merge and Close

When CodeRabbit status shows "Review completed" with state "success":

```bash
# Merge the PR (squash merge)
gh pr merge {pr_number} --squash --repo host-uk/{repo}

# The issue will auto-close if "Closes #N" was in PR body
# Otherwise, close manually:
gh issue close {number} --repo host-uk/{repo}
```

---

## Step 7: Restart Loop

After merging:

1. Remove PR from `PENDING_PRS`
2. Check remaining pending PRs for reviews
3. Pick up next issue
4. **Restart this skill** to continue the loop

```
>>> LOOP COMPLETE - Restart /php-agent to continue working <<<
```

---

## PHP Packages Reference

| Package | Type | Description |
|---------|------|-------------|
| core-php | foundation | Core framework - events, modules, lifecycle |
| core-tenant | module | Multi-tenancy, workspaces, users |
| core-admin | module | Admin panel, Livewire, Flux UI |
| core-api | module | REST API, webhooks |
| core-mcp | module | MCP server framework |
| core-agentic | module | AI agent orchestration |
| core-bio | product | Link-in-bio pages |
| core-social | product | Social media scheduling |
| core-analytics | product | Privacy-first analytics |
| core-commerce | module | Billing, Stripe |
| core-content | module | CMS, pages, blog |

---

## Troubleshooting

### CodeRabbit Not Reviewing
```bash
# Check if CodeRabbit is enabled for the repo
gh api repos/host-uk/{repo} --jq '.topics'

# Check webhook configuration
gh api repos/host-uk/{repo}/hooks
```

### Tests Failing
```bash
# Run with verbose output
composer test -- --verbose

# Run specific test
composer test -- --filter=TestClassName
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

## Labels Reference

- `lang:php` - PHP code changes
- `agent:ready` - Ready for AI agent pickup
- `good-first-issue` - Simple, well-defined tasks
- `priority:high` - Should be addressed soon
- `follow-up` - Created from another issue
- `needs:review` - Awaiting human review
