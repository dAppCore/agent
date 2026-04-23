---
name: openbrain-recall
description: Use when needing to retrieve prior knowledge from OpenBrain - the shared vector+keyword memory store. Triggered by phrases like "what did we decide about X", "recall from openbrain", "search my memory", "find the previous discussion about Y". Returns semantically-ranked memories with source + confidence.
---

Use this skill when Hermes should query OpenBrain before answering from scratch. The Python `OpenBrainMemoryProvider` exposes `brain_recall` directly, injects current workspace and org defaults when available, and returns the raw API payload so Hermes can recover prior decisions, bugs, plans, architecture notes, user guidance, or reference context without extra coaching.

## When to use

- Trigger on explicit recall phrases such as "what did we decide about the MemoryProvider plugin", "recall from OpenBrain", "search my memory for the previous discussion", or "find the earlier note about Hermes skills".
- Use for prior-context questions across user, feedback, project, and reference material: stable user instructions, earlier corrections, project decisions, architecture notes, bugs, research, documentation, or conventions.
- Run this before `openbrain-remember` when a near-duplicate might already exist, or when you need the UUID and current wording of an older memory before writing a superseding replacement.

## Tool contract

Call `brain_recall` with a single JSON object. Hermes' Python provider exposes top-level filter fields, not the nested `filter` object used by some other OpenBrain integrations.

```json
{
  "query": "required natural-language search query",
  "limit": 5,
  "top_k": 5,
  "workspace_id": 73,
  "org": "lthn",
  "project": "corepy",
  "type": ["decision", "architecture"],
  "keywords": ["memoryprovider", "hermes"],
  "boost_keywords": ["openbrain"],
  "agent_id": "codex",
  "min_confidence": 0.7
}
```

- `query` is required and must be a natural-language search string up to 2,000 characters.
- `limit` and `top_k` are both accepted. If `limit` is present and `top_k` is absent, the provider copies `limit` into `top_k` before sending the request.
- `workspace_id` is optional in the skill surface. If omitted, the provider injects the configured workspace automatically.
- `org` is optional. If the provider was initialised with an org, it injects that default when you do not pass one.
- `project`, `agent_id`, `keywords`, `boost_keywords`, and `min_confidence` are optional ranking/filter controls.
- `type` may be a single string or an array. Valid values are `fact`, `decision`, `observation`, `convention`, `research`, `plan`, `bug`, `architecture`, `documentation`, `service`, `pattern`, `context`, and `procedure`.
- Successful responses are raw API JSON with `status` plus a `data` object. In normal operation, inspect `data.count`, `data.memories`, and optionally `data.scores`.
- Each recalled memory typically includes `id`, `agent_id`, `type`, `content`, `tags`, `project`, `confidence`, `score`, `source`, `supersedes_id`, `supersedes_count`, `expires_at`, `deleted_at`, `created_at`, and `updated_at`.

## Example invocation

```json
{
  "tool": "brain_recall",
  "args": {
    "query": "what did we decide about the Hermes MemoryProvider plugin for OpenBrain?",
    "project": "corepy",
    "type": ["decision", "architecture"],
    "keywords": ["memoryprovider", "hermes"],
    "boost_keywords": ["openbrain"],
    "limit": 4,
    "min_confidence": 0.7
  }
}
```

Expected response shape:

```json
{
  "status": 200,
  "data": {
    "count": 2,
    "scores": {
      "550e8400-e29b-41d4-a716-446655440000": 1.5,
      "550e8400-e29b-41d4-a716-446655440111": 0.72
    },
    "memories": [
      {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "agent_id": "codex",
        "type": "decision",
        "content": "Use the MemoryProvider plugin to expose OpenBrain recall and remember tools to Hermes.",
        "tags": ["source:manual", "hermes", "openbrain", "memoryprovider"],
        "project": "corepy",
        "confidence": 0.95,
        "score": 1.5,
        "source": "manual",
        "supersedes_id": null,
        "supersedes_count": 0,
        "expires_at": null,
        "deleted_at": null,
        "created_at": "2026-04-23T11:30:00+00:00",
        "updated_at": "2026-04-23T11:30:00+00:00"
      }
    ]
  }
}
```

## When NOT to use

- Do not trigger this skill for general codebase search, repo grep, web research, or questions that can be answered from the visible conversation alone.
- Do not misfire on ambiguous phrases like "remember to fix this later" when the user is setting a task reminder rather than asking for shared-memory recall.
- Do not use this as a write path. If the user wants to persist a new decision, correction, preference, or lesson, switch to `openbrain-remember` after checking whether an equivalent memory already exists.

## Related skills

- `openbrain-remember` is the write companion. Recall first to avoid duplicate memories, to gather UUIDs for supersession, and to anchor new writes in the existing knowledge base.
