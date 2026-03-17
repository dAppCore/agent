# CODEX.md

Instructions for OpenAI Codex when working in the Core ecosystem.

## MCP Tools Available

You have access to core-agent MCP tools. Use them:

- `brain_recall` — Search OpenBrain for context about any package, pattern, or decision
- `brain_remember` — Store what you learn for other agents (Claude, Gemini, future LEM)
- `agentic_dispatch` — Dispatch tasks to other agents
- `agentic_status` — Check agent workspace status

**ALWAYS `brain_remember` significant findings** — your deep analysis of package internals, error patterns, security observations. This builds the shared knowledge base.

## Core Ecosystem Conventions

### Go Packages (forge.lthn.ai/core/*)

- **Error handling**: `coreerr.E("pkg.Method", "what failed", err)` from `go-log`. NEVER `fmt.Errorf` or `errors.New`.
  - Import as: `coreerr "forge.lthn.ai/core/go-log"`
  - Always 3 args: operation, message, cause (use `nil` if no cause)
  - `coreerr.E` returns `*log.Err` which implements `error` and `Unwrap()`

- **File I/O**: `coreio.Local.Read/Write/Delete/EnsureDir` from `go-io`. NEVER `os.ReadFile/WriteFile/MkdirAll`.
  - Import as: `coreio "forge.lthn.ai/core/go-io"`
  - Security: go-io validates paths, prevents traversal

- **Process management**: `go-process` for spawning external commands. Supports Timeout, GracePeriod, KillGroup.

- **UK English**: colour, organisation, centre, initialise (never American spellings)

- **Test naming**: `TestFoo_Good` (happy path), `TestFoo_Bad` (expected errors), `TestFoo_Ugly` (panics/edge cases)

- **Commits**: `type(scope): description` with `Co-Authored-By: Virgil <virgil@lethean.io>`

### PHP Packages (CorePHP)

- **Actions pattern**: Single-purpose classes with `use Action` trait, static `::run()` helper
- **Tenant isolation**: `BelongsToWorkspace` trait on ALL models with tenant data
- **Strict types**: `declare(strict_types=1)` in every file
- **Testing**: Pest syntax, not PHPUnit

## Review Focus Areas

When reviewing code, prioritise:

1. **Security**: Path traversal, injection, hardcoded secrets, unsafe input
2. **Error handling**: coreerr.E() convention compliance
3. **File I/O**: go-io usage, no raw os.* calls
4. **Tenant isolation**: BelongsToWorkspace on all tenant models (PHP)
5. **Test coverage**: Are critical paths tested?

## Training Data

Your reviews generate training data for LEM (our fine-tuned model). Be thorough and structured in your findings — every observation helps improve the next generation of reviews.
