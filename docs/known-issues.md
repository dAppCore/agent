<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Known Issues — core/agent

Accepted trade-offs and by-design behaviours that can surprise a caller. These are not bugs; they are documented so nobody re-reports them.

## By design

- **Bridge-mode recall/list return empty synchronously.** `pkg/brain/provider.go`'s HTTP recall and list handlers forward to the IDE bridge and return an empty result body; the real results arrive asynchronously over WebSocket. This only affects bridge-mode clients — the `DirectSubsystem` path (`pkg/brain/direct.go`) returns results inline.
- **`defaultBranch` fallback.** Auto-PR targets `dev` and falls back to `main` / `master` when `origin/HEAD` is unavailable. This covers effectively all repos in the ecosystem.

## Conventions to be aware of

- **`CODE_PATH` is interpreted in two ways.** `prep.go` treats `CODE_PATH` as the parent code directory (defaulting to `~/Code`), while some Forge tooling treats it as a repo root. Set it deliberately when overriding.
- **`core.Env("DIR_HOME")` is static at process init.** For test overrides use `CORE_HOME` rather than expecting `DIR_HOME` to change at runtime.
- **Monitor path helpers normalise separators.** API/glob output needs separator normalisation for cross-platform correctness — keep that in mind when adding new path-producing code in `pkg/monitor`.

## Test-infrastructure gaps

- `dispatch` / `review_queue` / `spawnAgent` have unit coverage but no full integration tests against a live runner — that needs process-mocking infrastructure.
- `drainQueue`'s more complex branches would benefit from tests with filesystem scaffolding.
