<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Local Inference Typologies

Measured on Apple Silicon M3 Ultra with 96 GB unified memory, using MLX VLM
OpenAI-compatible servers and Automatic Prefix Caching (APC).

This document is the operational map. Use `docs/local-inference.md` for launch
commands and lower-level runner notes.

## Decision Summary

Use one large foreground model for developer flow. Use small models for bounded
background work: PR interaction, writing, issue triage, cron jobs, summaries,
and tool-result compression.

| Workflow | Default | Interactive limit | Hard edge | Notes |
| --- | --- | ---: | ---: | --- |
| Developer coding | Qwen3.6 27B 4-bit | 1 | 1 active foreground | Best fit for the way this machine is used. |
| Developer synthesis | Gemma 4 26B-A4B 4-bit | 1 | 1 active foreground | Good alternative main lane; long-context full-window mix still needs more testing. |
| Xhigh reasoning | Gemma 4 31B 4-bit | 1 | 1 active foreground | Run alone until full-window APC behaviour is measured. |
| Helper/cron fast lane | Gemma 4 E2B 4-bit | 4 beside a big model | 8 beside Qwen | Do not run 12 beside Qwen; that crossed into crash territory. |
| Helper/cron quality lane | Gemma 4 E4B MXFP8 | 2 beside a big model | 4 beside Qwen | Better writing/review helper, lower concurrency. |

Qwen3.6 is marketed as a 256k-context model. The local MLX config reports the
exact limit as `262144` tokens.

## Safe Topologies

### One Big Developer Agent

Use this for the normal hands-on coding session.

| Lane | Model | Port | Context | Cache mode |
| --- | --- | ---: | ---: | --- |
| Main | `mlx-community/Qwen3.6-27B-4bit` | 8003 | 262144 | APC |

Launch:

```bash
scripts/qwen36_local_stack.py serve
```

Policy:

| Setting | Value |
| --- | --- |
| Active big agents | 1 |
| Helpers during cold prefill | 0 |
| Helpers after Qwen prefix is hot | 4 E2B default, 8 E2B max |
| Qwen fan-out | Avoid for interactive work |

### Big Qwen Plus E2B Helpers

Use this for background batches while keeping the Qwen coding lane hot.

| Lane | Model | Count | Context |
| --- | --- | ---: | ---: |
| Main | `mlx-community/Qwen3.6-27B-4bit` | 1 | 262144 |
| Helper | `mlx-community/gemma-4-e2b-it-4bit` | 4 default, 8 max | 131072 |

Observed safe mixed result:

| Shape | Result |
| --- | --- |
| 1 Qwen 128k cached + 8 E2B 128k cached | Passed, Qwen about 4.9s, E2B batch about 3.4s |
| 1 Qwen 128k cached + 12 E2B 128k cached | Unsafe; do not repeat |

Use E2B for short, bounded jobs: summarise PR comments, rewrite issue text,
classify inbox items, produce cron reports, compress logs, and prepare context
for the main model.

### Big Qwen Plus E4B Helpers

Use this when helper quality matters more than helper count.

| Lane | Model | Count | Context |
| --- | --- | ---: | ---: |
| Main | `mlx-community/Qwen3.6-27B-4bit` | 1 | 262144 |
| Helper | `mlx-community/gemma-4-e4b-it-mxfp8` | 2 default, 4 max | 131072 |

Observed safe mixed result:

| Shape | Result |
| --- | --- |
| 1 Qwen 128k cached + 4 E4B 128k cached | Passed, Qwen about 5.1s, E4B batch about 2.8s after cache warmup |

Use E4B for writing, careful summarisation, PR response drafting, and review
triage where small quality differences matter.

### Small-Model Batch Mode

Use this when the big foreground model is not running.

| Model | Interactive default | Observed hard edge | Notes |
| --- | ---: | ---: | --- |
| Gemma 4 E2B 4-bit | 8 at 128k | 16 at 128k, 17 OOM | Best background throughput lane. |
| Gemma 4 E4B MXFP8 | 4 at 128k | 9 at 128k, 10 latency cliff | Better helper quality, less headroom. |

The hard edge is not the working target. Use the interactive defaults unless a
cron batch can tolerate slowdowns and failure recovery.

## Measured Capacity

### Qwen3.6 27B 4-bit

| Prompt tokens | Concurrent requests | Latency | Peak memory | Result |
| ---: | ---: | ---: | ---: | --- |
| 63342 | 1 cold | 198.9s | 30.1 GB | First 64k prefill |
| 63342 | 1 cached | 2.3s | 34.0 GB | Exact APC hit |
| 126622 | 1 cold | 516.2s | 49.8 GB | First 128k prefill |
| 126622 | 1 cached | 2.0s | 51.2 GB | Exact APC hit |
| 126622 | 2 cached | 3.9s | 60.8 GB | Passed |
| 126622 | 3 cached | 10.3s | 68.1 GB | Passed, not normal workflow |
| 126622 | 4 cached | failed | n/a | Metal OOM |

Qwen APC was excellent for exact byte-stable repeats. It did not reuse a
previous 64k prefix when the prompt expanded to 128k, so design the harness
around exact stable prefixes rather than assuming partial-prefix reuse.

### Gemma 4 E2B and E4B Helpers

| Model | Prompt tokens | Concurrent requests | Batch latency | Peak memory | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| E2B 4-bit | 123804 | 1 cold | 26.1s | 12.0 GB | Cold prefill |
| E2B 4-bit | 123804 | 1 cached | 0.7s | 12.0 GB | Exact APC hit |
| E2B 4-bit | 123804 | 16 cached | 9.3s | 69.5 GB | Passed alone |
| E2B 4-bit | 123804 | 17 cached | failed | n/a | OOM |
| E4B MXFP8 | 128031 | 1 cold | 60.2s | 22.7 GB | Cold prefill |
| E4B MXFP8 | 128031 | 1 cached | 3.1s | 22.7 GB | Exact APC hit |
| E4B MXFP8 | 128031 | 8 cached | 11.0s | 69.4 GB | Passed alone |
| E4B MXFP8 | 123804 | 9 cached | 11.4s | 77.8 GB | Practical upper bound alone |
| E4B MXFP8 | 123804 | 10 cached | 68.4s | 77.8 GB | Latency cliff |

### Gemma 4 Main Lane

| Model | Prompt tokens | Cold latency | Cached latency | Peak memory | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| Gemma 4 26B-A4B 4-bit | 63430 | 41.5s | 1.0s | 22.8 GB | Passed |
| Gemma 4 E4B MXFP8 | 63426 | 23.1s | 1.1s | 14.7 GB | Passed beside 26B resident |

Treat Gemma 4 26B and 31B as one-at-a-time foreground models until their
full-window helper mix has been measured separately.

## Scheduling Rules

Use these defaults in CoreAgent or OpenCode harness policy.

```yaml
foreground:
  max_big_agents: 1
  preferred_coding_model: qwen36-27b
  allow_helpers_during_cold_prefill: false

helpers:
  default_model: gemma4-e2b
  default_count_with_big_agent: 4
  max_count_with_qwen27: 8
  e4b_default_count_with_big_agent: 2
  e4b_max_count_with_qwen27: 4

limits:
  qwen27_cached_fanout: 3
  qwen27_cached_fanout_for_interactive_work: 1
  e2b_alone_cached_fanout: 16
  e4b_alone_cached_fanout: 9
  forbidden_mixed_shape: qwen27_plus_12_e2b
```

## Cache Rules

APC is the feature that makes local agentic inference workable.

Keep these byte-stable:

| Prefix region | Notes |
| --- | --- |
| System prompt | Do not inject timestamps or per-run IDs. |
| Tool schema | Prefer a compact CoreAgent tool proxy over huge OpenCode schemas. |
| Repository summary | Stable file ordering and deterministic formatting. |
| AGENTS.md and policy text | Keep at the front of the prompt. |
| Previous state summary | Replace in fixed slots; avoid growing unbounded. |

Append only volatile content: the current user request, the current tool trace,
and the new diff or command output. Use the same `X-APC-Tenant` for related
requests.

Do not combine APC and MLX VLM KV quantisation in the same lane. TurboQuant is a
separate capacity experiment because APC is skipped when `--kv-bits` is active.

## Runner Guidance

| Runner | Use now | Reason |
| --- | --- | --- |
| MLX VLM | Yes | Working OpenAI-compatible server, APC, Qwen/Gemma tool parsers. |
| MLX LM | Maybe | Simpler text server, but not the measured APC path here. |
| vLLM Metal | Not for this workflow yet | Qwen/Gemma MTP paths exist upstream, but Metal validation was not stable enough for this Mac workflow. |
| llama.cpp | Optional GGUF fallback | Useful for simple local chat, not the measured full-window APC topology. |

Qwen3.6 has MTP metadata in the model config. Use that as a future vLLM/SGLang
validation track, not as a requirement for the current Metal workflow.

## Do Not Repeat

These settings crossed the useful boundary:

| Shape | Outcome |
| --- | --- |
| 4 cached 128k Qwen 27B requests | Metal OOM |
| 1 Qwen 27B plus 12 E2B helpers | Unsafe system-level stress |
| 10 cached 128k E4B helper requests alone | Latency cliff |
| 17 cached 128k E2B helper requests alone | OOM |

The practical workstation shape is one big model plus a small number of helpers,
not a maximum-throughput inference server.

