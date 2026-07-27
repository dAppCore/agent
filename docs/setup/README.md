<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Workspace setup

`setup` gets a repo ready to be worked by an agent: it detects the project type and
scaffolds a `.core/` directory. (For wiring the GitHub App, see
[`github-app.md`](github-app.md).)

## What it does

1. **Detects the project type** — Go, PHP, Node, Wails, … (`ProjectType`), from the
   files present.
2. **Scaffolds `.core/`** with the build + test contracts:
   - `.core/build.yaml` — how to build this project
   - `.core/test.yaml` — how to test it
3. Optionally **extracts a workspace template** from the embedded [library](../lib/)
   (`default`, `review`, or `security`) via `lib.ExtractWorkspace`.

The `.core/` contract is what lets dispatch/QA build and test any repo uniformly — the
runner reads `build.yaml`/`test.yaml` rather than guessing per-language commands.

## Checking it

`core-agent check` reports the workspace root and whether `agents.yaml` is present — the
quickest "is this repo set up?" probe.

## Next

[lib](../lib/) (the templates `setup` extracts) · [`github-app.md`](github-app.md)
(GitHub App) · [dispatch](../dispatch/) (consumes the `.core/` contract).
