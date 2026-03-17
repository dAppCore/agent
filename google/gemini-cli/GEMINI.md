# GEMINI.md

Instructions for Google Gemini CLI when working in the Core ecosystem.

## MCP Tools Available

You have access to core-agent MCP tools via the extension. Use them:

- `brain_recall` — Search OpenBrain for context about any package, pattern, or decision
- `brain_remember` — Store what you learn for other agents (Claude, Codex, future LEM)
- `agentic_dispatch` — Dispatch tasks to other agents
- `agentic_status` — Check agent workspace status

**ALWAYS `brain_remember` significant findings** — your analysis of patterns, conventions, security observations. This builds the shared knowledge base that all agents read.

## Core Ecosystem Conventions

### Go Packages (forge.lthn.ai/core/*)

- **Error handling**: `coreerr.E("pkg.Method", "what failed", err)` from `go-log`. NEVER `fmt.Errorf`.
  - Import as: `coreerr "forge.lthn.ai/core/go-log"`
  - Always 3 args: operation, message, cause (use `nil` if no cause)

- **File I/O**: `coreio.Local.Read/Write/Delete/EnsureDir` from `go-io`. NEVER `os.ReadFile`.
  - Import as: `coreio "forge.lthn.ai/core/go-io"`

- **UK English**: colour, organisation, centre, initialise

- **Test naming**: `TestFoo_Good`, `TestFoo_Bad`, `TestFoo_Ugly`

- **Commits**: `type(scope): description` with `Co-Authored-By: Virgil <virgil@lethean.io>`

### PHP Packages (CorePHP)

- **Actions pattern**: `use Action` trait, static `::run()` helper
- **Tenant isolation**: `BelongsToWorkspace` on ALL tenant models
- **Strict types**: `declare(strict_types=1)` everywhere

## Your Role

You are best used for:
- **Fast batch operations** — convention sweeps, i18n, docs
- **Lightweight coding** — small fixes, boilerplate, test generation
- **Quick audits** — file scans, pattern matching

Leave deep security review to Codex and complex architecture to Claude.

## Training Data

Your work generates training data for LEM. Be consistent with conventions — every file you touch should follow the patterns above perfectly.
