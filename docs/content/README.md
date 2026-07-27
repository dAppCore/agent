<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Content & training

Two adjacent things live here: generating content through AI providers, and gathering
agent output into training data.

## Content generation

Generate content via a provider (`claude`, …) and track it as a batch:

| Verb / func | What it does |
|-------------|--------------|
| `content/batch` (`ContentBatchGenerate`) | kick off a batch generation — returns a `batch_id`; supports dry-run |
| `content/from-plan` (`ContentFromPlan`) | generate from a [plan](../plans/) (`plan_slug`), merging the prompt-template payload |
| `content/status` (`ContentStatus`) | poll a batch by `batch_id` for `status` + `content` |

A result is a `ContentResult{Provider, Model, Content}`. Providers are validated before
the call (an unknown/unavailable provider is rejected up front, not mid-batch).

## Training data

The training side gathers agent findings + outputs into training data that feeds the LEM
training pipeline (agent work → datasets). This is the "agents produce their own training
signal" loop — what an agent did on a dispatch can become a future training example.

## Next

[plans](../plans/) (`content/from-plan` source) · [pipeline](../pipeline/) (findings that
feed training).
