---
name: architecture-review
description: Stage 4 of review pipeline — dispatch Backend Architect agent to check lifecycle events, Actions pattern, tenant isolation, and namespace mapping
---

# Architecture Review Stage

Dispatch the **Backend Architect** agent to review code changes for architectural correctness.

## When to Use

Invoked as Stage 4 of `/review:pipeline`. Can be run standalone via `/review:pipeline --stage=architecture`.

## Agent Persona

Read the Backend Architect persona from:
```
agents/engineering/engineering-backend-architect.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Dispatch a subagent:

```
[Persona content from engineering-backend-architect.md]

## Your Task

Review the following code changes for architectural correctness. This is a READ-ONLY review.

### Changed Files
[List of changed files]

### Diff
[Full diff content]

### Check These Patterns

1. **Lifecycle Events**: Are modules using `$listens` arrays in Boot.php? Are routes registered via event callbacks (`$event->routes()`), not direct `Route::get()` calls?

2. **Actions Pattern**: Is business logic in Action classes with `use Action` trait? Or is it leaking into controllers/Livewire components?

3. **Tenant Isolation**: Do new/modified models that hold tenant data use `BelongsToWorkspace`? Are migrations adding `workspace_id` with foreign key and cascade delete?

4. **Namespace Mapping**: Do files follow `src/Core/` → `Core\`, `src/Mod/` → `Core\Mod\`, `app/Mod/` → `Mod\`?

5. **Go Services** (if applicable): Are services registered via factory functions? Using `ServiceRuntime[T]`? Implementing `Startable`/`Stoppable`?

6. **Dependency Direction**: Do changes respect the dependency graph? Products depend on core-php and core-tenant, never on each other.

### Output Format

## Architecture Review

### Lifecycle Events
[Findings or "Correct — events used properly"]

### Actions Pattern
[Findings or "Correct — logic in Actions"]

### Tenant Isolation
[Findings or "Correct — BelongsToWorkspace on all tenant models"]

### Namespace Mapping
[Findings or "Correct"]

### Dependency Direction
[Findings or "Correct"]

### Issues
- **VIOLATION**: file:line — [Description]
- **WARNING**: file:line — [Description]
- **SUGGESTION**: file:line — [Description]

**Summary**: X violations, Y warnings, Z suggestions
```

3. Return the subagent's review as the stage output
