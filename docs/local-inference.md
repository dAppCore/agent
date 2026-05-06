<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Local Inference

CoreAgent can dispatch OpenCode against local OpenAI-compatible endpoints with
`opencode:<profile>`. The profile only tells OpenCode which endpoint and model
name to use; the model server still has to be launched separately.

## Chatter

Use `lthn/lemer-mlx-bf16` as the small local chatter model. Run it as a
separate server from Gemma MTP; a Gemma MTP drafter is dimension-matched to the
target Gemma model and cannot be reused for Lemer.

```bash
/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server \
  --model lthn/lemer-mlx-bf16 \
  --host 127.0.0.1 \
  --port 8007 \
  --max-kv-size 32768 \
  --max-tokens 512
```

Dispatch with:

```bash
core agentic dispatch --agent opencode:lemer --repo core/agent --task "..."
```

Aliases: `opencode:lemer`, `opencode:lemer-chatter`, `opencode:chatter`.

`lthn/lemer-mlx-bf16` is verified through the MLX VLM OpenAI-compatible server.
The smaller `lthn/lemer-mlx` quantized checkpoint still needs separate loader
validation before it should be used as the HTTP chatter server.

## Gemma 4 on Metal

MLX-backed Gemma profiles use `core-mlx` provider names and expect MLX servers
on fixed local ports:

| Profile | Port | Model |
| --- | ---: | --- |
| `opencode:gemma4-mlx-agentic` | 8001 | `mlx-community/gemma-4-26b-a4b-it-4bit` |
| `opencode:gemma4-mlx-xhigh` | 8002 | `mlx-community/gemma-4-31b-it-4bit` |
| `opencode:gemma4-mlx-e2b` | 8004 | `mlx-community/gemma-4-e2b-it-4bit` |
| `opencode:gemma4-mlx-e4b` | 8005 | `mlx-community/gemma-4-e4b-it-mxfp8` |
| `opencode:gemma4-mlx-mtp` | 8010 | `mlx-community/gemma-4-26b-a4b-it-4bit` |
| `opencode:gemma4-mlx-xhigh-mtp` | 8011 | `mlx-community/gemma-4-31b-it-4bit` |

Example:

```bash
/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server \
  --model mlx-community/gemma-4-26b-a4b-it-4bit \
  --host 127.0.0.1 \
  --port 8001 \
  --max-kv-size 32768 \
  --max-tokens 2048
```

Gemma 4 MTP on MLX is exposed through the MLX VLM drafter path. The current PyPI
wheel tested as `mlx-vlm==0.4.4` did not expose `--draft-model`; install from
the Git repository until PyPI has the MTP release:

```bash
UV_CACHE_DIR=/private/tmp/uv-cache uv venv /private/tmp/core-agent-mlx-vlm --python 3.12
UV_CACHE_DIR=/private/tmp/uv-cache uv pip install \
  --python /private/tmp/core-agent-mlx-vlm/bin/python \
  --upgrade git+https://github.com/Blaizzy/mlx-vlm.git
```

For the 26B MoE agentic lane:

```bash
/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server \
  --host 127.0.0.1 \
  --port 8010 \
  --model mlx-community/gemma-4-26b-a4b-it-4bit \
  --draft-model mlx-community/gemma-4-26B-A4B-it-assistant-bf16 \
  --draft-kind mtp \
  --draft-block-size 3 \
  --kv-bits 3.5 \
  --kv-quant-scheme turboquant \
  --max-kv-size 32768 \
  --max-tokens 2048
```

Dispatch with `opencode:gemma4-mlx-mtp`.

For the 31B dense xhigh lane:

```bash
/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server \
  --host 127.0.0.1 \
  --port 8011 \
  --model mlx-community/gemma-4-31b-it-4bit \
  --draft-model mlx-community/gemma-4-31B-it-assistant-bf16 \
  --draft-kind mtp \
  --draft-block-size 3 \
  --kv-bits 3.5 \
  --kv-quant-scheme turboquant \
  --max-kv-size 32768 \
  --max-tokens 4096
```

Dispatch with `opencode:gemma4-mlx-xhigh-mtp`.

Raw OpenAI-compatible requests should disable thinking with the top-level
`enable_thinking` field:

```bash
curl http://127.0.0.1:8010/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "mlx-community/gemma-4-26b-a4b-it-4bit",
    "messages": [{"role": "user", "content": "Reply with exactly two words: metal ready"}],
    "max_tokens": 32,
    "temperature": 0,
    "enable_thinking": false
  }'
```

OpenCode currently reaches the MLX VLM server when the model key keeps the
Hugging Face namespace (`core-mlx/mlx-community/...`). A full edit smoke did not
complete without request-body injection, because OpenCode does not send
`enable_thinking:false`; use a request proxy or a non-thinking chatter endpoint
for harness work until that is wired through.

Single-request Metal measurements on the M3 Ultra 96GB:

| Model | MTP | Draft block | Generation tok/s | Peak memory |
| --- | --- | ---: | ---: | ---: |
| Gemma 4 E2B BF16 | off | - | 95.4 | 10.30 GB |
| Gemma 4 E2B BF16 | on | 6 | 76.0 | 10.46 GB |
| Gemma 4 26B-A4B 4-bit | off | - | 102.5 | 15.76 GB |
| Gemma 4 26B-A4B 4-bit | on | 3 | 125.1 | 16.58 GB |
| Gemma 4 31B 4-bit | off | - | 33.9 | 18.98 GB |
| Gemma 4 31B 4-bit | on | 3 | 43.3 | 19.73 GB |

For this machine, start with `--draft-block-size 3` on 26B and 31B. Block 6 is
the upstream single-request default, but it was slower on the tested 26B and
roughly flat on 31B. E2B is already fast enough that MTP overhead loses on short
decodes.

### Long Context and Prefix Cache

For agentic work, optimise the prefill path before tuning decode speed. OpenCode
can add about 29k input tokens before task-specific context, so repeated
128k-window turns need prefix caching more than they need short-prompt MTP
microbenchmarks.

MLX VLM git builds expose Automatic Prefix Caching (APC). Use APC when multiple
turns or agents share the same stable prefix:

```bash
APC_ENABLED=1 \
APC_NUM_BLOCKS=10000 \
APC_BLOCK_SIZE=16 \
APC_LAYER_MAJOR_MEMORY_MIN_TOKENS=50000 \
APC_DISK_PATH=/private/tmp/mlx-vlm-apc \
APC_DISK_MAX_GB=8 \
APC_DISK_SHARD_MAX_BLOCKS=256 \
/private/tmp/core-agent-mlx-vlm/bin/mlx_vlm.server \
  --host 127.0.0.1 \
  --port 8020 \
  --model mlx-community/gemma-4-e4b-it-mxfp8 \
  --max-kv-size 131072 \
  --max-tokens 256
```

Send the same `X-APC-Tenant` header for requests that should share cached
prefixes. Keep the system prompt, repository summary, AGENTS.md content, tool
schema, and long context byte-stable; append only the changing user request and
tool trace suffix. Do not enable MLX VLM `--kv-bits` on the APC lane: APC is
skipped when KV-cache quantisation is enabled, so run a separate TurboQuant lane
for resident-context capacity testing.

Near-128k APC measurements on the M3 Ultra 96GB, using MLX VLM git
`0.5.0`, OpenAI-compatible chat requests, `temperature=0`, and `max_tokens=64`:

| Model | Concurrent agents | Prompt tokens | Batch latency | Peak memory | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| E4B MXFP8 | 1 cold | 128031 | 60.2s | 22.7 GB | Cold prefill baseline |
| E4B MXFP8 | 1 cached | 128031 | 3.1s | 22.7 GB | Full APC hit |
| E4B MXFP8 | 4 cached | 128031 | 5.9s | 38.8 GB | Usable |
| E4B MXFP8 | 8 cached | 123804 | 11.0s | 69.4 GB | Usable |
| E4B MXFP8 | 9 cached | 123804 | 11.4s | 77.8 GB | Practical upper bound |
| E4B MXFP8 | 10 cached | 123804 | 68.4s | 77.8 GB | Latency cliff |
| E2B 4-bit | 1 cold | 123804 | 26.1s | 12.0 GB | Cold prefill baseline |
| E2B 4-bit | 1 cached | 123804 | 0.7s | 12.0 GB | Full APC hit |
| E2B 4-bit | 16 cached | 123804 | 9.3s | 69.5 GB | Usable |
| E2B 4-bit | 17 cached | 123804 | failed | OOM | Metal out of memory |

Use these as scheduler defaults:

| Lane | Recommended full-window agents | Hard cap observed | Notes |
| --- | ---: | ---: | --- |
| E4B chatter/router | 8 | 9 | Ten completed but was too slow for interactive agent work. |
| E2B chatter/router | 16 | 16 | Seventeen crashed the MLX VLM process after a BatchRotatingKVCache error path. |

For E2B and E4B MTP, the MLX community assistant cards recommend
`--draft-block-size 6` for single requests and `--draft-block-size 3` for
batched generation. Treat block 3 as the default for OpenCode-style concurrent
agent traffic.

## Gemma 4 MTP on ROCm

Use vLLM for the ROCm lane when you want Gemma 4 tool calling, reasoning
parsing, and MTP speculative decoding behind one OpenAI-compatible API:

```bash
vllm serve google/gemma-4-26B-A4B-it \
  --host 127.0.0.1 \
  --port 8008 \
  --max-model-len 32768 \
  --kv-cache-dtype turboquant_k8v4 \
  --enable-auto-tool-choice \
  --tool-call-parser gemma4 \
  --reasoning-parser gemma4 \
  --chat-template examples/tool_chat_template_gemma4.jinja \
  --speculative-config '{"model":"gg-hf-am/gemma-4-26B-it-assistant","num_speculative_tokens":4}'
```

Dispatch with `opencode:gemma4-vllm-mtp`.

For the 31B dense xhigh lane:

```bash
vllm serve google/gemma-4-31B-it \
  --host 127.0.0.1 \
  --port 8009 \
  --max-model-len 32768 \
  --kv-cache-dtype turboquant_k8v4 \
  --enable-auto-tool-choice \
  --tool-call-parser gemma4 \
  --reasoning-parser gemma4 \
  --chat-template examples/tool_chat_template_gemma4.jinja \
  --speculative-config '{"model":"gg-hf-am/gemma-4-31B-it-assistant","num_speculative_tokens":4}'
```

Dispatch with `opencode:gemma4-vllm-xhigh-mtp`.

TurboQuant presets are selected through vLLM's `--kv-cache-dtype` flag. Start
with `turboquant_k8v4` because it keeps FP8 keys and 4-bit values; the vLLM
docs report about 2.6x KV compression with the smallest perplexity hit of the
TurboQuant presets. Only move to `turboquant_4bit_nc` or lower-bit presets
after quality checks pass for the target workflow.

vLLM automatically skips the first and last two layers for TurboQuant boundary
protection. Extra skips can be added with `--kv-cache-dtype-skip-layers`, for
example when keeping sliding-window layers native is faster on a target GPU.
