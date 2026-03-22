# Known Issues — core/agent

Accepted issues from 7 rounds of Codex review. These are acknowledged
trade-offs or enhancement requests, not bugs.

## API Enhancements (brain/direct.go)

- `direct.go:134` — `remember` drops `confidence`, `supersedes`, `expires_in` from `RememberInput`. Standalone clients can't set persistence metadata.
- `direct.go:153` — `recall` never forwards `filter.min_confidence`. Direct-mode recall can't apply confidence cutoff.
- `direct.go:177` — `recall` drops API-returned tags, only synthesises `source:*`. Callers lose real memory tags.
- `provider.go:303` — `list` forwards `limit` as query-string value instead of integer. REST path diverges from MCP contract.

## Test Coverage Gaps

- `pkg/lib` has no dedicated tests for template extraction or embedded prompt/task loading.
- `dispatch`/`review_queue`/`spawnAgent` have no integration tests. Need test infrastructure for process mocking.
- `drainQueue` complex logic has no unit tests with filesystem scaffolding.

## Conventions

- `defaultBranch` falls back to `main`/`master` when `origin/HEAD` unavailable. Acceptable — covers 99% of repos.
- `CODE_PATH` interpreted differently by `syncRepos` (repo root) vs rest of tooling (`CODE_PATH/core`). Known inconsistency.

## Async Bridge Returns (brain/provider.go)

- `provider.go:247` — recall HTTP handler forwards to bridge but returns empty `RecallOutput`. Results arrive async via WebSocket — by design for the IDE bridge path.
- `provider.go:297` — list HTTP handler same pattern. Only affects bridge-mode clients, not DirectSubsystem.

## Compile Issues

- `pkg/setup` doesn't compile — calls `lib.RenderFile`, `lib.ListDirTemplates`, `lib.ExtractDir` which don't exist yet. Package is not imported by anything.

## Changelog

- 2026-03-21: Created from 7 rounds of Codex static review
- 2026-03-21: Updated after 9 total rounds (77+ findings, 73+ fixed, 4 false positives)
