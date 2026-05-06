<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Local Inference

CoreAgent can dispatch OpenCode against local OpenAI-compatible endpoints with
`opencode:<profile>`. The profile only tells OpenCode which endpoint and model
name to use; the model server still has to be launched separately.

## Chatter

Use `lthn/lemer-mlx-bf16` as the small local chatter model:

```bash
mlx_lm.server \
  --model lthn/lemer-mlx-bf16 \
  --host 127.0.0.1 \
  --port 8007 \
  --chat-template-args '{"enable_thinking":false}' \
  --decode-concurrency 1 \
  --prompt-concurrency 1
```

Dispatch with:

```bash
core agentic dispatch --agent opencode:lemer --repo core/agent --task "..."
```

Aliases: `opencode:lemer`, `opencode:lemer-chatter`, `opencode:chatter`.

`lthn/lemer-mlx` is the smaller quantized checkpoint, but the current
`mlx_lm` loader rejects its quantization tensors as extra parameters. Direct
generation with `lthn/lemer-mlx-bf16` works on Metal; the quantized checkpoint
needs the Gemma4 VLM loader path before it can be used as the HTTP chatter
server.

Current local `mlx_lm.server` on Python 3.14 also crashes OpenAI chat requests
inside the generation thread with `There is no Stream(gpu, 0) in current
thread`. Treat the MLX server profiles as endpoint contracts; use direct
`mlx_lm.generate` for benchmarking until the MLX server thread issue is fixed.

## Gemma 4 on Metal

MLX-backed Gemma profiles use `core-mlx` provider names and expect MLX servers
on fixed local ports:

| Profile | Port | Model |
| --- | ---: | --- |
| `opencode:gemma4-mlx-agentic` | 8001 | `mlx-community/gemma-4-26b-a4b-it-4bit` |
| `opencode:gemma4-mlx-xhigh` | 8002 | `mlx-community/gemma-4-31b-it-4bit` |
| `opencode:gemma4-mlx-e2b` | 8004 | `mlx-community/gemma-4-e2b-it-4bit` |
| `opencode:gemma4-mlx-e4b` | 8005 | `mlx-community/gemma-4-e4b-it-mxfp8` |

Example:

```bash
mlx_lm.server \
  --model mlx-community/gemma-4-26b-a4b-it-4bit \
  --host 127.0.0.1 \
  --port 8001 \
  --chat-template-args '{"enable_thinking":false}' \
  --decode-concurrency 1 \
  --prompt-concurrency 1
```

Gemma 4 MTP on MLX is currently exposed through the MLX VLM drafter path rather
than this OpenAI-compatible server profile. Use it for direct benchmarking:

```bash
python -m mlx_vlm generate \
  --model mlx-community/gemma-4-26B-A4B-it-bf16 \
  --draft-model mlx-community/gemma-4-26B-A4B-it-assistant-bf16 \
  --draft-kind mtp \
  --draft-block-size 6 \
  --prompt "Explain speculative decoding in 3 sentences." \
  --max-tokens 256 \
  --temperature 0
```

## Gemma 4 MTP on ROCm

Use vLLM for the ROCm lane when you want Gemma 4 tool calling, reasoning
parsing, and MTP speculative decoding behind one OpenAI-compatible API:

```bash
vllm serve google/gemma-4-26B-A4B-it \
  --host 127.0.0.1 \
  --port 8008 \
  --max-model-len 32768 \
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
  --enable-auto-tool-choice \
  --tool-call-parser gemma4 \
  --reasoning-parser gemma4 \
  --chat-template examples/tool_chat_template_gemma4.jinja \
  --speculative-config '{"model":"gg-hf-am/gemma-4-31B-it-assistant","num_speculative_tokens":4}'
```

Dispatch with `opencode:gemma4-vllm-xhigh-mtp`.
