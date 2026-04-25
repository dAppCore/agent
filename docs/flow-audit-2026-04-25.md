<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Flow Library Audit - 2026-04-25

## Summary

This audit used `/Users/snider/Code/host-uk/core/plans/code/core/agent/flow/RFC.md` as the source of truth.

- YAML flows present in `pkg/lib/flow/`: `2`
- Canonical YAML flows mandated by RFC section 3.1: `15`
- Canonical YAML flows missing from `pkg/lib/flow/`: `13`
- Additional RFC example-only path not present in section 3.1: `pr/merge.yaml` (missing, spec ambiguity)

Current state in one sentence: only `upgrade/v080-plan.yaml` and `upgrade/v080-implement.yaml` exist, while every other RFC library subdirectory is absent, and the executable runner does not yet implement the RFC flow model.

## RFC Baseline

RFC section 3.1 defines this canonical library under `pkg/lib/flow/`:

- `deploy/from/forge.yaml`
- `deploy/to/forge.yaml`
- `deploy/to/github.yaml`
- `implement/security-scan.yaml`
- `implement/upgrade-deps.yaml`
- `pr/to-dev.yaml`
- `pr/to-main.yaml`
- `upgrade/v080-plan.yaml`
- `upgrade/v080-implement.yaml`
- `verify/go-qa.yaml`
- `verify/php-qa.yaml`
- `workspace/prepare/go.yaml`
- `workspace/prepare/php.yaml`
- `workspace/prepare/ts.yaml`
- `workspace/prepare/devops.yaml`
- `workspace/prepare/secops.yaml`

The RFC gate example in section 5.3 also references `pr/merge.yaml`, but that path is not listed in the canonical section 3.1 layout. I have treated it as an example-only extra and listed it separately below.

## YAML Inventory

Every YAML file currently present in `pkg/lib/flow/`, grouped by subdirectory:

- `upgrade/`
  - `pkg/lib/flow/upgrade/v080-implement.yaml`
  - `pkg/lib/flow/upgrade/v080-plan.yaml`

Non-YAML content currently present at the top level of `pkg/lib/flow/`:

- Markdown files: `cpp.md`, `docker.md`, `git.md`, `go.md`, `npm.md`, `php.md`, `prod-push-polish.md`, `py.md`, `release.md`, `ts.md`
- Go code: `flow.go`, `flow_test.go`
- Misc: `upgrade/README.md`

These top-level Markdown files are legacy embedded assets, but they do not satisfy the RFC's path-addressed YAML library.

## Per-Subdirectory Matrix

| RFC subdirectory | RFC-required YAMLs | Present on disk | Status | Notes |
|---|---:|---:|---|---|
| `deploy/` | 3 | 0 | Missing | `deploy/` does not exist. |
| `implement/` | 2 | 0 | Missing | `implement/` does not exist. |
| `pr/` | 2 | 0 | Missing | `pr/` does not exist. RFC section 5.3 also references `pr/merge.yaml`. |
| `upgrade/` | 2 | 2 | Present | Both RFC upgrade YAMLs exist. They do not match the executable `cmd`-only parser contract. |
| `verify/` | 2 | 0 | Missing | `verify/` does not exist. |
| `workspace/prepare/` | 5 | 0 | Missing | `workspace/` and `workspace/prepare/` do not exist. |

## Library / Parser Alignment

The library exists on disk, but the parser and embedded lookup paths are not aligned with the RFC.

### Findings

1. `pkg/lib/flow/flow.go:16` embeds only `*.md` and `upgrade/`, not the full RFC directory tree.
2. `pkg/lib/flow/flow.go:25` defines a `Step` schema with only `name`, `cmd`, `args`, and `continueOnError`.
3. `pkg/lib/flow/flow.go:101` validates that every step must provide `cmd`.
4. The existing upgrade YAMLs do not use `cmd` steps. They use fields such as `description`, `commands`, `verify`, `commit`, `source`, `section`, `scope`, `pattern`, `output`, and `sections`.
5. `pkg/lib/flow/flow_test.go:152` already acknowledges this mismatch: `TestFlow_LoadEmbedded_Good` skips if no embedded flow matches the current `cmd`-only contract.
6. `pkg/lib/lib.go:24` embeds `all:flow`, but `pkg/lib/lib.go:194` still resolves embedded flows as `slug + ".md"` only. That means the mounted embedded flow FS cannot resolve RFC-style YAML paths such as `upgrade/v080-plan`.

### Consequence

Even the two YAML files that exist are not executable under the current `pkg/lib/flow` parser contract, and the mounted embedded library path resolution is still Markdown-slug based instead of RFC path-addressed YAML based.

## Runner Feature Matrix

| Feature | RFC expectation | Source evidence | Observed behaviour | Status |
|---|---|---|---|---|
| Embedded path-addressed YAML lookup | `run flow` should resolve embedded RFC paths like `upgrade/v080-plan.yaml` | `pkg/lib/lib.go:194` loads only `slug + ".md"`; `pkg/agentic/commands.go:1090` calls `lib.Flow(flowSlugFromPath(path))` | `./core-agent run/flow upgrade/v080-plan --dry-run` exits `1` and errors on `flow/v080-plan.md` | Missing |
| `flow:` directive | Runner should resolve and execute nested flows recursively | `pkg/agentic/commands.go:1178` resolves nested flows in preview; `pkg/agentic/flow.go:118` rejects nested `flow` execution with `cannot execute nested flow references` | Preview resolves; execution path rejects | Preview-only / missing in execution |
| `when:` conditional steps | Runner should evaluate conditions before executing a step | `pkg/agentic/commands.go:1054` declares `When`, but no execution path reads `step.When` | No source evidence of evaluation; no preview rendering either | Missing |
| `parallel:` fan-out | Runner should execute fan-out branches | `pkg/agentic/commands.go:1058` declares `Parallel`; `pkg/agentic/commands.go:1199` prints `parallel:` in preview; `pkg/agentic/flow.go:143` executes a simple sequential loop only | Preview can print branches; execution never runs them | Preview-only / missing in execution |
| `--dry-run` | `run flow ... --dry-run` should show what would execute | `pkg/agentic/flow.go:32` maps `dry-run` to `runFlowCommand` preview mode | Works for preview output; does not validate executable semantics | Present, but preview-only |

## Dry-Run Probe

### Command used

```bash
./core-agent run/flow pkg/lib/flow/upgrade/v080-plan.yaml --dry-run
```

### Exit code

`0`

### Stdout shape

The checked-in `core-agent` binary printed:

- startup logs from `brain` and `monitor`
- `flow:  pkg/lib/flow/upgrade/v080-plan.yaml`
- `dry-run: true`
- `name:  v0.8.0 Upgrade Plan`
- `desc:  Generate UPGRADE.md for a Go package - audit banned imports, test naming, usage comments`
- `steps: 5`
- numbered step names:
  - `1. audit-deps`
  - `2. audit-imports`
  - `3. audit-tests`
  - `4. audit-comments`
  - `5. write-plan`

Notably, the output contained no execution summary, no command dispatch, and no validation of the step schema. This behaves as a preview path, not as an executable runner dry-run with RFC semantics.

### Additional probes

```bash
./core-agent run/flow upgrade/v080-plan --dry-run
```

- Exit code: `1`
- Result: fails with `flow not found` because it looks for `flow/v080-plan.md`

```bash
./core-agent run/flow go --dry-run
```

- Exit code: `0`
- Result: resolves `embedded:go` and prints `content: 241 chars`
- Interpretation: embedded Markdown slug lookup works, embedded RFC YAML path lookup does not

### Note on runtime vs source

The checked-in binary behaved like preview mode for both `run/flow` and `flow/preview`, even without `--dry-run`. Current source in `pkg/agentic/flow.go` still contains an execution path, so treat the binary output above as observational evidence from the local artifact, and the feature matrix above as the authoritative source audit.

## Child Ticket List

One ticket per missing RFC flow YAML:

1. `feat(agent/flow): add deploy/from/forge.yaml`
2. `feat(agent/flow): add deploy/to/forge.yaml`
3. `feat(agent/flow): add deploy/to/github.yaml`
4. `feat(agent/flow): add implement/security-scan.yaml`
5. `feat(agent/flow): add implement/upgrade-deps.yaml`
6. `feat(agent/flow): add pr/to-dev.yaml`
7. `feat(agent/flow): add pr/to-main.yaml`
8. `feat(agent/flow): add verify/go-qa.yaml`
9. `feat(agent/flow): add verify/php-qa.yaml`
10. `feat(agent/flow): add workspace/prepare/go.yaml`
11. `feat(agent/flow): add workspace/prepare/php.yaml`
12. `feat(agent/flow): add workspace/prepare/ts.yaml`
13. `feat(agent/flow): add workspace/prepare/devops.yaml`
14. `feat(agent/flow): add workspace/prepare/secops.yaml`

Runner / library feature tickets needed before the RFC flow library can actually execute as specified:

15. `feat(agent/flow): load embedded RFC YAML flows by path instead of Markdown slug lookup`
16. `feat(agent/flow): align executable flow schema with RFC YAML step fields`
17. `feat(agent/flow): execute nested flow: directives in run/flow`
18. `feat(agent/flow): evaluate when: conditional steps in run/flow`
19. `feat(agent/flow): execute parallel: fan-out steps in run/flow`

Spec-reconciliation ticket for the extra RFC example path:

20. `feat(agent/flow): add pr/merge.yaml or remove the RFC section 5.3 reference`

## Recommended Dispatch Order

This order unblocks the most downstream consumers first.

1. Land the runner / library foundation tickets first:
   - `feat(agent/flow): load embedded RFC YAML flows by path instead of Markdown slug lookup`
   - `feat(agent/flow): align executable flow schema with RFC YAML step fields`
   - `feat(agent/flow): execute nested flow: directives in run/flow`
   - `feat(agent/flow): evaluate when: conditional steps in run/flow`
   - `feat(agent/flow): execute parallel: fan-out steps in run/flow`
2. Add the lowest-level reusable leaf flows next:
   - `verify/go-qa.yaml`
   - `verify/php-qa.yaml`
   - `workspace/prepare/go.yaml`
   - `workspace/prepare/php.yaml`
   - `workspace/prepare/ts.yaml`
   - `workspace/prepare/devops.yaml`
   - `workspace/prepare/secops.yaml`
   - `pr/to-dev.yaml`
   - `pr/to-main.yaml`
3. Add composed flows that depend on those leaf flows:
   - `implement/security-scan.yaml`
   - `implement/upgrade-deps.yaml`
4. Add deploy flows after the core composition model is stable:
   - `deploy/from/forge.yaml`
   - `deploy/to/forge.yaml`
   - `deploy/to/github.yaml`
5. Resolve the RFC ambiguity around `pr/merge.yaml` last unless a consumer already depends on the gate example.

## Bottom Line

- The RFC calls for a 15-flow canonical YAML library; only 2 of those flows exist.
- The only populated RFC subdirectory is `upgrade/`.
- `flow:`, `when:`, and executable `parallel:` support are not implemented in the runner.
- `run/flow --dry-run` works as a preview of an on-disk YAML file, but not as proof that RFC-style flows are executable.
- Embedded RFC YAML path lookup is also missing; the current embedded path still resolves Markdown slugs instead of the RFC directory structure.
