# v0.8.0 Upgrade Flow

Two-stage flow for upgrading Go packages to Core v0.8.0 AX compliance.

## Stage 1: Plan (`v080-plan.yaml`)
Dispatched with: `agentic_dispatch repo=X task="plan" template=upgrade/v080-plan`

Produces UPGRADE.md with:
- Dependency audit (current version → v0.8.0-alpha.1)
- Banned import findings (file:line → Core replacement)
- Test naming violations (file:line → suggested rename)
- Missing usage-example comments (file:line)

## Stage 2: Implement (`v080-implement.yaml`)
Dispatched with: `agentic_dispatch repo=X task="implement" template=upgrade/v080-implement`

Reads UPGRADE.md and implements changes in order:
1. Upgrade deps → verify build
2. Fix production imports → verify build
3. Fix test imports → verify tests
4. Rename tests → verify tests
5. Add comments → verify vet

Each step commits separately with conventional commit format.

## Usage
```
# Plan first
agentic_dispatch repo=go-cache task="Create v0.8.0 upgrade plan" agent=codex branch=dev

# Then implement
agentic_dispatch repo=go-cache task="Implement v0.8.0 upgrade from UPGRADE.md" agent=codex branch=dev
```
