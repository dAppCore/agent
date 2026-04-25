# OpenBrain Implementation Plan — DEPRECATED / MOVED

**STATUS**: Superseded 2026-04-23. The authoritative OpenBrain RFC is now `plans/project/lthn/ai/RFC-OPENBRAIN.md` in the host-uk/core/plans tree.

## Why this file still exists
Historical reference only. Left in place so git blame resolves and so links in older PRs / notes don't 404. Do NOT implement against this file.

## What changed
The pre-redesign implementation plan assumed: single Qdrant collection, nomic-embed-text embeddings, synchronous embedding on write. The current implementation model is: scoped collections, embeddinggemma 768-dim, async embedding via the EmbedMemory job + Elasticsearch integration for tag/full-text search.

## What to read instead
plans/project/lthn/ai/RFC-OPENBRAIN.md — the single source of truth.
