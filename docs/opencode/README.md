<!-- SPDX-License-Identifier: EUPL-1.2 -->
# OpenCode plugin

OpenCode is one of the dispatch runners (a **native, host** runner — see
[dispatch](../dispatch/)). It runs against OpenAI-compatible endpoints — typically the
local `lthn-mlx` serve — so you can dispatch work to a local model instead of a cloud
provider.

## Dispatching to OpenCode

Use an `opencode:<profile>` agent string:

```
agentic_dispatch(repo, task="…", agent="opencode:gemma4-mlx-agentic", branch="dev")
```

The part after the colon is the **profile**, which tells OpenCode *which endpoint and
model* to use. The model server still has to be running separately (see
[inference](../inference/)).

## Profiles

Profiles are **kv-backed** and managed over the hub's loopback HTTP control plane
(`core-agent hub`):

| Method + path | Does |
|---------------|------|
| `GET /profile` | list profiles (a `default` is seeded) |
| `GET /profile/<name>` | get one |
| `POST /profile` | create/save (`{"name":"…"}`) |
| `DELETE /profile/<name>` | delete |

## Listing dispatch models

```
core-agent opencode-models
```

Lists the OpenCode dispatch models the host's `opencode` sees — the **free Zen** tier and
the **authed Go** tiers.

## Next

[dispatch](../dispatch/) (how runners are chosen) · [inference](../inference/) (the local
endpoints OpenCode targets) · [cli](../cli/) (`hub`, `opencode-models`).
