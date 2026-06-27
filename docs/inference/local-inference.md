<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Local Inference

CoreAgent can dispatch OpenCode against local OpenAI-compatible endpoints with
`opencode:<profile>`. The profile only tells OpenCode which endpoint and model
name to use; the model server still has to be launched separately.

For workstation sizing and safe model combinations, start with
[`typologies.md`](typologies.md).

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

### Gemma 4 Agentic Stack

For the current Apple Silicon lane, prefer no-MTP MLX VLM with APC:

| Lane | Runner | Model | Default port | Context | Purpose |
| --- | --- | --- | ---: | ---: | --- |
| Main | MLX VLM | `mlx-community/gemma-4-26b-a4b-it-4bit` | 8001 | 262144 | Planning, synthesis, final edits, long-lived project context |
| Helper | MLX VLM | `mlx-community/gemma-4-e4b-it-mxfp8` | 8005 | 131072 | Sub-agent work, file/tool investigation, summaries back to main |

Launch both with:

```bash
scripts/gemma4_local_stack.py serve
```

Show the exact commands without launching:

```bash
scripts/gemma4_local_stack.py serve --dry-run
```

Show CoreAgent/OpenCode profile overrides:

```bash
scripts/gemma4_local_stack.py opencode-env
```

Check health and APC counters:

```bash
scripts/gemma4_local_stack.py status
```

The helper can be switched to E2B for higher concurrency:

```bash
scripts/gemma4_local_stack.py serve --helper helper-e2b
```

For one-off helper prompts, `scripts/local-agent.sh` wraps the same local
profiles and adds a bounded project-context preamble:

```bash
scripts/local-agent.sh --profile gemma-helper "summarise the current failure"
scripts/local-agent.sh --profile gemma-main "draft the final implementation plan"
```

It also has Qwen3.6 lanes pre-wired for OpenAI-compatible servers:

```bash
scripts/local-agent.sh --profile qwen36 --dry-run "review the qwen lane"
scripts/local-agent.sh --profile qwen36-moe --dry-run "review the qwen moe lane"
```

Use `--file-limit` or `LOCAL_FILE_LIMIT` to control how many source-file paths
are included in the prompt. The default is 800 paths.

### Qwen3.6 Coding Stack

For coding on Apple Silicon, use `mlx-community/Qwen3.6-27B-4bit` as the
preferred Qwen lane. It is denser than the 35B-A3B MoE lane, better aligned to
coding work, and still fits the M3 Ultra 96GB at 262k context.

| Lane | Runner | Model | Default port | Context | Purpose |
| --- | --- | --- | ---: | ---: | --- |
| Coding | MLX VLM | `mlx-community/Qwen3.6-27B-4bit` | 8003 | 262144 | Main coding and review lane |
| Coding MXFP8 | MLX VLM | `mlx-community/Qwen3.6-27B-mxfp8` | 8006 | 262144 | Quality-first coding lane to validate next |
| MoE helper | MLX VLM | `mlx-community/Qwen3.6-35B-A3B-4bit` | 8008 | 262144 | Optional throughput/helper lane |

Launch the default APC lane:

```bash
scripts/qwen36_local_stack.py serve
```

Show commands without launching:

```bash
scripts/qwen36_local_stack.py serve --dry-run
scripts/qwen36_local_stack.py serve --lane moe35 --dry-run
scripts/qwen36_local_stack.py serve --mode turboquant --dry-run
```

Use APC for agentic turns that can keep an exact byte-stable prefix. Use the
TurboQuant mode as a separate capacity experiment because MLX VLM does not use
APC when KV quantisation is enabled.

Measured `mlx-community/Qwen3.6-27B-4bit` APC behaviour on the M3 Ultra 96GB:

| Prompt tokens | Concurrent agents | Latency | APC result | Peak memory | Notes |
| ---: | ---: | ---: | --- | ---: | --- |
| 21 | 1 cold | 1.0s | none | 16.6 GB | Functional smoke, `enable_thinking=false` |
| 63342 | 1 cold | 198.9s | none | 30.1 GB | First 64k prefill |
| 63342 | 1 cached | 2.3s | exact hit, 63326 tokens | 34.0 GB | Byte-stable repeat |
| 126622 | 1 cold | 516.2s | no partial 64k reuse | 49.8 GB | First 128k prefill |
| 126622 | 1 cached | 2.0s | exact hit, 126606 tokens | 51.2 GB | Byte-stable repeat |
| 126622 | 2 cached | 3.9s | exact hits | 60.8 GB | Good full-window pair |
| 126622 | 3 cached | 10.3s | disk exact hits | 68.1 GB | Practical full-window cap |
| 126622 | 4 cached | failed | Metal OOM | n/a | `kIOGPUCommandBufferCallbackErrorOutOfMemory` |

Current scheduler default: allow one Qwen3.6-27B main agent at 128k, allow up to
three only for cached full-window fan-out, and run additional helpers on Gemma
E2B/E4B unless a smaller Qwen helper is validated.

Qwen3.6 MTP is present in the model config (`mtp_num_hidden_layers=1`) and in
vLLM's Qwen3.5/Qwen3.6 MTP model paths. Treat it as a vLLM/SGLang validation
track for now. The tested Metal path for real work is MLX VLM with APC; the
Gemma assistant-drafter MTP path is not reusable for Qwen.

Tool execution should stay in the harness layer, such as CoreAgent or OpenCode.
MLX VLM gives the local OpenAI-compatible chat endpoints and APC behaviour; the
harness owns file reads, edits, shell commands, permissioning, and summarising
helper results back into the main lane. This keeps the main context smaller and
keeps the model servers free of large tool-schema prompts when a thinner
CoreAgent tool proxy can do the routing.

No-MTP APC measurements with both lanes resident on the M3 Ultra 96GB:

| Lane | Prompt tokens | Cold latency | Cached latency | APC match | Peak memory |
| --- | ---: | ---: | ---: | ---: | ---: |
| Main 26B-A4B 4-bit | 63430 | 41.5s | 1.0s | 63414 | 22.8 GB |
| Helper E4B MXFP8 | 63426 | 23.1s | 1.1s | 63410 | 14.7 GB |

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
