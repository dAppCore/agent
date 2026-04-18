# RFC Plan — How to Start a core/agent Session

> For future Claude sessions. Do this FIRST before touching code.

## Step 1: Load the Domain

Read these files in order using ReadFile. Yes, all of them. The ~2000 tokens of boot cost pays for itself immediately — zero corrections, zero rediscovery.

```
1. ReadFile /Users/snider/Code/core/go/docs/RFC.md          (1278 lines — core/go contract, 21 sections)
2. ReadFile /Users/snider/Code/core/agent/docs/RFC.md        (~500 lines — core/agent contract, 22 sections)
3. ReadFile /Users/snider/Code/core/go-process/docs/RFC.md   (~224 lines — go-process contract, 8 sections)
```

After loading all three, you have the full domain model:
- Every core/go primitive and how core/agent uses it
- The current state of core/agent (what's migrated, what isn't)
- The file layout with per-file migration actions
- The quality gates (10 disallowed imports, test naming, string concat)
- The completion pipeline architecture
- The entitlement/permission model

## Step 2: Verify Context

After loading, you should be able to answer without looking at code:
- What does `c.Action("agentic.dispatch").Run(ctx, opts)` do?
- How do direct `s.Core().Process()` calls replace the old process wrapper layer?
- What replaces the ACTION cascade in `handlers.go`?
- Which imports are disallowed and what replaces each one?
- What does `c.Entitled("agentic.concurrency", 1)` check?

If you can't answer these, re-read the RFCs.

## Step 3: Work the Migration

The core/agent RFC Section "Current State" has the annotated file layout. Each file is marked DELETE, REWRITE, or MIGRATE with the specific action.

Priority order:
1. `OnStartup`/`OnShutdown` return `Result` (breaking, do first)
2. Replace `unsafe.Pointer` → `Fs.NewUnrestricted()` (paths.go)
3. Replace `os.WriteFile` → `Fs.WriteAtomic` (status.go)
4. Replace `core.ValidateName` / `core.SanitisePath` (prep.go, plan.go)
5. Replace `core.ID()` (plan.go)
6. Register capabilities as named Actions (OnStartup)
7. Replace ACTION cascade with Task pipeline (handlers.go)
8. Use `s.Core().Process()` directly in call sites. The old `proc.go` wrapper layer has been removed.
9. AX-7 test rename + gap fill
10. Example tests per source file

## Step 4: Session Cadence

Follow the CLAUDE.md session cadence:
- **0-50%**: Build — implement the migration
- **50%**: Feature freeze — finish what's in progress
- **60%+**: Refine — review passes on RFC.md, docs, CLAUDE.md, llm.txt
- **80%+**: Save state — update RFCs with what shipped

## What NOT to Do

- Don't guess the architecture — it's in the RFCs
- Don't use `os`, `os/exec`, `fmt`, `errors`, `io`, `path/filepath`, `encoding/json`, `strings`, `log`, `unsafe` — Core has primitives for all of these
- Don't use string concat with `+` — use `core.Concat()` or `core.Path()`
- Don't add `fmt.Println` — use `core.Println()`
- Don't write anonymous closures in command registration — extract to named methods
- Don't nest `c.ACTION()` calls — use `c.Task()` composition
