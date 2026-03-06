---
name: pr
description: Review a pull request
args: <pr-number>
---

# PR Review

Review a GitHub pull request.

## Usage

```
/review:pr 123
/review:pr 123 --security
/review:pr 123 --quick
```

## Process

1. **Fetch PR details**
   ```bash
   gh pr view 123 --json title,body,author,files,additions,deletions
   ```

2. **Get PR diff**
   ```bash
   gh pr diff 123
   ```

3. **Check CI status**
   ```bash
   gh pr checks 123
   ```

4. **Review changes**
   - Correctness
   - Security (if --security)
   - Tests coverage
   - Documentation

5. **Provide feedback**

## Output Format

```markdown
## PR Review: #123 - Add user authentication

**Author**: @username
**Files**: 5 changed (+120, -30)
**CI**: ✓ All checks passing

### Summary
Brief description of what this PR does.

### Review

#### Approved ✓
- Clean implementation
- Good test coverage
- Documentation updated

#### Changes Requested ✗
- **src/auth.go:42** - Missing input validation
- **src/auth.go:87** - Error not handled

#### Comments
- Consider adding rate limiting
- Nice use of middleware pattern

---
**Recommendation**: Approve with minor changes
```

## Actions

After review, you can:
```bash
# Approve
gh pr review 123 --approve

# Request changes
gh pr review 123 --request-changes --body "See comments"

# Comment only
gh pr review 123 --comment --body "Looks good overall"
```
