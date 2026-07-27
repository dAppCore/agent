<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Runners — native vs containerised

A dispatch resolves which runner to use from the `agent` string, and *where* it runs.
This is the detail behind [dispatch](README.md).

## Where each runner runs

| Runner | Location |
|--------|----------|
| `claude`, `coderabbit`, `opencode` | **on the host** (native) |
| `codex`, `gemini` | **inside a container** |

Native runners need the tool installed on the machine; containerised runners are isolated
so an untrusted change can't touch the host.

## The agent string — `provider[:model]`

The provider picks the runner; the optional model after the colon is passed through:

- `codex:gpt-5.4-mini`, `claude:opus`, `opencode:gemma4-mlx-agentic`
- bare `codex` uses the provider default.

For containerised runners the model is passed to the agent as `--model`.

## Container runtimes

`containerCommandFor` supports three runtimes, with the `core-dev` image and an optional
GPU flag:

| Runtime | Binary |
|---------|--------|
| `RuntimeDocker` | `docker` |
| `RuntimeApple` | Apple Virtualization (VZ) |
| `RuntimePodman` | `podman` |

**An unknown or empty runtime name falls back to `docker`** (`containerRuntimeBinary`), so
a misconfigured runtime never silently breaks dispatch. The agent runs `exec` in the
workspace mounted at `/ws`.

See also [shell](../shell/) to attach a terminal to one of these containers.
