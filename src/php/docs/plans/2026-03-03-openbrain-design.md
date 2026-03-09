# OpenBrain Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Shared vector-indexed knowledge store that all agents (Virgil, Charon, Darbs, LEM) read/write through MCP, building singular state across sessions.

**Architecture:** MariaDB for relational metadata + Qdrant for vector embeddings. Four MCP tools in php-agentic. Go bridge in go-ai for CLI agents. Ollama for embedding generation.

**Repos:** `forge.lthn.ai/core/php-agentic` (primary), `forge.lthn.ai/core/go-ai` (bridge)

---

## Problem

Agent knowledge is scattered:
- Virgil's `MEMORY.md` files in `~/.claude/projects/*/memory/` — file-based, single-agent, no semantic search
- Plans in `docs/plans/` across repos — forgotten after completion
- Session handoff notes in `agent_sessions.handoff_notes` — JSON blobs, not searchable
- Research findings lost when context windows compress

When Charon discovers a scoring calibration bug, Virgil only knows about it if explicitly told. There's no shared knowledge graph.

## Concept

**OpenBrain** — "Open" means open protocol (MCP), not open source. All agents on the platform access the same knowledge graph via `brain_*` MCP tools. Data is stored *for agents* — structured for near-native context transfer between sessions and models.

## Data Model

### `brain_memories` table (MariaDB)

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key, also Qdrant point ID |
| `workspace_id` | FK | Multi-tenant isolation |
| `agent_id` | string | Who wrote it (virgil, charon, darbs, lem) |
| `type` | enum | `decision`, `observation`, `convention`, `research`, `plan`, `bug`, `architecture` |
| `content` | text | The knowledge (markdown) |
| `tags` | JSON | Topic tags for filtering |
| `project` | string nullable | Repo/project scope (null = cross-project) |
| `confidence` | float | 0.0–1.0, how certain the agent is |
| `supersedes_id` | UUID nullable | FK to older memory this replaces |
| `expires_at` | timestamp nullable | TTL for session-scoped context |
| `deleted_at` | timestamp nullable | Soft delete |
| `created_at` | timestamp | |
| `updated_at` | timestamp | |

### `openbrain` Qdrant collection

- **Vector dimension:** 768 (nomic-embed-text via Ollama)
- **Distance metric:** Cosine
- **Point ID:** MariaDB UUID
- **Payload:** `workspace_id`, `agent_id`, `type`, `tags`, `project`, `confidence`, `created_at` (for filtered search)

## MCP Tools

### `brain_remember` — Store a memory

```json
{
  "content": "LEM emotional_register was blind to negative emotions. Fixed by adding 8 weighted pattern groups.",
  "type": "bug",
  "tags": ["scoring", "emotional-register", "lem"],
  "project": "eaas",
  "confidence": 0.95,
  "supersedes": "uuid-of-outdated-memory"
}
```

Agent ID injected from MCP session context. Returns the new memory UUID.

**Pipeline:**
1. Validate input
2. Embed content via Ollama (`POST /api/embeddings`, model: `nomic-embed-text`)
3. Insert into MariaDB
4. Upsert into Qdrant with payload metadata
5. If `supersedes` set, soft-delete the old memory and remove from Qdrant

### `brain_recall` — Semantic search

```json
{
  "query": "How does verdict classification work?",
  "top_k": 5,
  "filter": {
    "project": "eaas",
    "type": ["decision", "architecture"],
    "min_confidence": 0.5
  }
}
```

**Pipeline:**
1. Embed query via Ollama
2. Search Qdrant with vector + payload filters
3. Get top-K point IDs with similarity scores
4. Hydrate from MariaDB (content, tags, supersedes chain)
5. Return ranked results with scores

Only returns latest version of superseded memories (includes `supersedes_count` so agent knows history exists).

### `brain_forget` — Soft-delete or supersede

```json
{
  "id": "uuid",
  "reason": "Superseded by new calibration approach"
}
```

Sets `deleted_at` in MariaDB, removes point from Qdrant. Keeps audit trail.

### `brain_list` — Browse (no vectors)

```json
{
  "project": "eaas",
  "type": "decision",
  "agent_id": "charon",
  "limit": 20
}
```

Pure MariaDB query. For browsing, auditing, bulk export. No embedding needed.

## Architecture

### PHP side (`php-agentic`)

```
Mcp/Tools/Agent/Brain/
├── BrainRemember.php
├── BrainRecall.php
├── BrainForget.php
└── BrainList.php

Services/
└── BrainService.php      # Ollama embeddings + Qdrant client + MariaDB CRUD

Models/
└── BrainMemory.php       # Eloquent model

Migrations/
└── XXXX_create_brain_memories_table.php
```

`BrainService` handles:
- Ollama HTTP calls for embeddings
- Qdrant REST API (upsert, search, delete points)
- MariaDB CRUD via Eloquent
- Supersession chain management

### Go side (`go-ai`)

Thin bridge tools in the MCP server that proxy `brain_*` calls to Laravel via the existing WebSocket bridge. Same pattern as `ide_chat_send` / `ide_session_create`.

### Data flow

```
Agent (any Claude)
    ↓ MCP tool call
Go MCP server (local, macOS/Linux)
    ↓ WebSocket bridge
Laravel php-agentic (lthn.ai, de1)
    ↓                    ↓
MariaDB              Qdrant
(relational)         (vectors)
    ↑
Ollama (embeddings)
```

PHP-native agents skip the Go bridge — call `BrainService` directly.

### Infrastructure

- **Qdrant:** New container on de1. Shared between OpenBrain and EaaS scoring (different collections).
- **Ollama:** Existing instance. `nomic-embed-text` model for 768d embeddings. CPU is fine for the volume (~10K memories).
- **MariaDB:** Existing instance on de1. New table in the agentic database.

## Integration

### Plans → Brain

On plan completion, agents can extract key decisions/findings and `brain_remember` them. Optional — agents decide what's worth persisting. The plan itself stays in `agent_plans`; lessons learned go to the brain.

### Sessions → Brain

Handoff notes (summary, next_steps, blockers) can auto-persist as memories with `type: observation` and optional TTL. Agents can also manually remember during a session.

### MEMORY.md migration

Seed data: collect all `MEMORY.md` files from `~/.claude/projects/*/memory/` across worktrees. Parse into individual memories, embed, and load into OpenBrain. After migration, `brain_recall` replaces file-based memory.

### EaaS

Same Qdrant instance, different collection (`eaas_scoring` vs `openbrain`). Shared infrastructure, separate concerns.

### LEM

LEM models query the brain for project context during training data curation or benchmark analysis. Same MCP tools, different agent ID.

## What this replaces

- Virgil's `MEMORY.md` files (file-based, single-agent, no search)
- Scattered `docs/plans/` findings that get forgotten
- Manual "Charon found X" cross-agent handoffs
- Session-scoped knowledge that dies with context compression

## What this enables

- Any Claude picks up where another left off — semantically
- Decisions surface when related code is touched
- Knowledge graph grows with every session across all agents
- Near-native context transfer between models and sessions
