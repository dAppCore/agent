---
name: openbrain-remember
description: Use when needing to persist durable knowledge into OpenBrain - the shared vector+keyword memory store. Triggered by phrases like "remember this decision", "save this for later", "store in openbrain", "keep this in memory for future sessions". Persists scoped memories with tags, confidence, and optional supersession or expiry.
---

Use this skill when Hermes should write a distilled, durable memory that future sessions can recall. The Python `OpenBrainMemoryProvider` exposes `brain_remember` directly; the goal is to save stable knowledge with explicit confidence and scope, not to dump raw chat history or transient scratch notes.

## When to use

- Trigger on explicit write phrases such as "remember this", "store this in OpenBrain", "save this decision for later", or "keep this in shared memory for future sessions".
- Use for durable user guidance, feedback, project knowledge, or reference notes: user preferences and standing instructions, corrections or review outcomes, project decisions and architecture, reusable procedures, bugs, plans, research, or documentation.
- Recall before writing when duplication is possible. If the new memory replaces an older one, look up the current record first and write with `supersedes` instead of creating conflicting copies.

## Tool contract

Call `brain_remember` with a single JSON object matching the Python provider surface:

```json
{
  "content": "required durable memory text",
  "type": "decision",
  "tags": ["hermes", "openbrain"],
  "project": "corepy",
  "org": "lthn",
  "confidence": 0.96,
  "supersedes": "550e8400-e29b-41d4-a716-446655440000",
  "expires_in": 72
}
```

- `content` and `type` are required. `content` may be up to 50,000 characters, but store the distilled fact, not the entire transcript.
- Valid `type` values are `fact`, `decision`, `observation`, `convention`, `research`, `plan`, `bug`, `architecture`, `documentation`, `service`, `pattern`, `context`, and `procedure`.
- `tags`, `project`, `confidence`, `supersedes`, and `expires_in` are optional. The provider also exposes `org`; it is passed through when present, but workspace/auth context still comes from the loaded MemoryProvider session rather than a `workspace_id` argument.
- `confidence` must be between `0.0` and `1.0`. If omitted, the backend defaults to `0.8`.
- `supersedes` must be the UUID of an older memory in the same workspace. Use it only when the new entry genuinely replaces the prior one.
- `expires_in` is the number of hours until expiry. Use it for short-lived context instead of polluting long-term memory.
- `source` is output-only. Do not invent a `source` field in the request; the stored memory will usually come back with `source: "manual"` unless the backend set something else.
- There is no separate `memory_type` field for `user`, `feedback`, `project`, or `reference`. Treat those as routing buckets and map them onto the real OpenBrain fields:
  - `user`: stable preferences or standing instructions. Usually `type: fact` or `context`, with a `user` tag if useful.
  - `feedback`: corrections, rejected approaches, review findings, or lessons from mistakes. Usually `type: decision`, `bug`, or `convention`, with a `feedback` tag.
  - `project`: repo-specific architecture, plans, procedures, services, or conventions. Usually `type: architecture`, `plan`, `procedure`, `service`, `pattern`, or `decision`, and set `project`.
  - `reference`: distilled external docs, research, or source material worth reusing. Usually `type: documentation` or `research`, with a `reference` tag.
- Confidence discipline matters more than verbosity:
  - `0.95-1.0`: explicit user instruction, final decision, or directly verified fact.
  - `0.8-0.94`: strong project knowledge or a confirmed fix worth keeping by default.
  - `0.6-0.79`: useful but somewhat provisional context; consider expiry if it may stale quickly.
  - Below `0.6`: avoid unless you clearly mark uncertainty and there is real future value.
- Successful responses are raw API JSON with `status` plus a `data` object containing the created memory, typically including `id`, `agent_id`, `type`, `content`, `tags`, `project`, `confidence`, `score`, `source`, `supersedes_id`, `supersedes_count`, `expires_at`, `deleted_at`, `created_at`, and `updated_at`.

## Example invocation

```json
{
  "tool": "brain_remember",
  "args": {
    "content": "Hermes should auto-discover OpenBrain recall and remember tools through the MemoryProvider plugin once ticket #73 is wired. The matching skills live under hermes/skills/openbrain-recall and hermes/skills/openbrain-remember.",
    "type": "decision",
    "tags": ["hermes", "openbrain", "memoryprovider", "ticket:75"],
    "project": "corepy",
    "confidence": 0.96
  }
}
```

Expected response shape:

```json
{
  "status": 201,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440222",
    "agent_id": "codex",
    "type": "decision",
    "content": "Hermes should auto-discover OpenBrain recall and remember tools through the MemoryProvider plugin once ticket #73 is wired. The matching skills live under hermes/skills/openbrain-recall and hermes/skills/openbrain-remember.",
    "tags": ["hermes", "openbrain", "memoryprovider", "ticket:75"],
    "project": "corepy",
    "confidence": 0.96,
    "score": 0,
    "source": "manual",
    "supersedes_id": null,
    "supersedes_count": 0,
    "expires_at": null,
    "deleted_at": null,
    "created_at": "2026-04-23T11:45:00+00:00",
    "updated_at": "2026-04-23T11:45:00+00:00"
  }
}
```

## When NOT to use

- Do not store ephemeral chatter, raw logs, full transcripts, scratchpad thoughts, or one-off reminders like "remember to run tests later".
- Do not write a new memory if `openbrain-recall` already returns an equivalent fact unless you are deliberately superseding or refining it.
- Do not persist secrets, tokens, passwords, private personal data, or unverified guesses framed as settled fact.

## Related skills

- `openbrain-recall` is the read companion. Use it before writing to dedupe, to fetch the UUID for `supersedes`, and after important writes when you need to confirm the new durable memory is what future recalls should surface.
