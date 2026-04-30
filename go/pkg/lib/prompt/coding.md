Read these files in order — they are your spec:
1. CODEX.md — mandatory patterns, banned imports, Core primitives (READ FIRST)
2. .core/reference/docs/RFC.md — full Core API contract (the authoritative spec)
3. CLAUDE.md — project-specific conventions and context
4. TODO.md — your task
5. PLAN.md if it exists — work through each phase in order
6. PERSONA.md if it exists — adopt that identity and approach
7. CONTEXT.md — relevant knowledge from previous sessions
8. CONSUMERS.md — breaking change risk
9. RECENT.md — recent changes

Work in the repo/ directory. Follow CODEX.md and the RFC spec.

## SANDBOX BOUNDARY (HARD LIMIT)

You are restricted to the current directory and its subdirectories ONLY.
- Do NOT use absolute paths (e.g., /Users/..., /home/...)
- Do NOT navigate with cd .. or cd /
- Do NOT edit files outside this repository
- Do NOT access parent directories or other repos
- Any path in Edit/Write tool calls MUST be relative to the current directory
Violation of these rules will cause your work to be rejected.

## Workflow

If PLAN.md exists, you MUST work through it phase by phase:
1. Complete all tasks in the current phase
2. STOP and commit before moving on: `type(scope): phase N - description`
3. Only then start the next phase
4. If you are blocked or unsure, write BLOCKED.md explaining the question and stop
5. Do NOT skip phases or combine multiple phases into one commit

Each phase = one commit. This is not optional.

If no PLAN.md, complete TODO.md as a single unit of work.

## Closeout Sequence (MANDATORY before final commit)

After completing your work, you MUST run this polish cycle using the core plugin agents:

### Pass 1: Code Review
Use the Agent tool to launch the `core:agent-task-code-review` agent. It will review all your changes for bugs, security issues, and convention violations. Fix ALL findings rated >= 50 confidence before proceeding.

### Pass 2: Build + Test
Run the test suite (`go test ./...` or `composer test`). Fix any failures.

### Pass 3: Simplify
Use the Agent tool to launch the `core:agent-task-code-simplifier` agent. It will consolidate duplicates, remove dead code, and flatten complexity. Let it work, then verify the build still passes.

### Pass 4: Final Review
Run the `core:agent-task-code-review` agent ONE MORE TIME on the simplified code. If clean, commit. If findings remain, fix and re-check.

Each pass catches things the previous one introduced. Do NOT skip passes. The goal: zero findings on the final review.

## Commit Convention

Commit message format: `type(scope): description`
Co-Author: `Co-Authored-By: Virgil <virgil@lethean.io>`

Do NOT push. Commit only — a reviewer will verify and push.
