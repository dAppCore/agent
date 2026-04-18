---
title: MCP Tools Reference
description: Complete reference for core-agentic MCP tools
updated: 2026-01-29
---

# MCP Tools Reference

This document provides a complete reference for all MCP tools in the `core-agentic` package.

## Overview

Tools are organised into categories:

| Category | Description | Tools Count |
|----------|-------------|-------------|
| plan | Work plan management | 5 |
| phase | Phase operations | 3 |
| session | Session tracking | 8 |
| state | Persistent state | 3 |
| task | Task completion | 2 |
| template | Plan templates | 3 |
| content | Content generation | 6 |

## Plan Tools

### plan_create

Create a new work plan with phases and tasks.

**Scopes:** `write`

**Input:**
```json
{
  "title": "string (required)",
  "slug": "string (optional, auto-generated)",
  "description": "string (optional)",
  "context": "object (optional)",
  "phases": [
    {
      "name": "string",
      "description": "string",
      "tasks": ["string"]
    }
  ]
}
```

**Output:**
```json
{
  "success": true,
  "plan": {
    "slug": "my-plan-abc123",
    "title": "My Plan",
    "status": "draft",
    "phases": 3
  }
}
```

**Dependencies:** workspace_id in context

---

### plan_get

Get a plan by slug with full details.

**Scopes:** `read`

**Input:**
```json
{
  "slug": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "plan": {
    "slug": "my-plan",
    "title": "My Plan",
    "status": "active",
    "progress": {
      "total": 5,
      "completed": 2,
      "percentage": 40
    },
    "phases": [...]
  }
}
```

---

### plan_list

List plans with optional filtering.

**Scopes:** `read`

**Input:**
```json
{
  "status": "string (optional: draft|active|completed|archived)",
  "limit": "integer (optional, default 20)"
}
```

**Output:**
```json
{
  "success": true,
  "plans": [
    {
      "slug": "plan-1",
      "title": "Plan One",
      "status": "active"
    }
  ],
  "count": 1
}
```

---

### plan_update_status

Update a plan's status.

**Scopes:** `write`

**Input:**
```json
{
  "slug": "string (required)",
  "status": "string (required: draft|active|completed|archived)"
}
```

---

### plan_archive

Archive a plan with optional reason.

**Scopes:** `write`

**Input:**
```json
{
  "slug": "string (required)",
  "reason": "string (optional)"
}
```

## Phase Tools

### phase_get

Get phase details by plan slug and phase order.

**Scopes:** `read`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "phase_order": "integer (required)"
}
```

---

### phase_update_status

Update a phase's status.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "phase_order": "integer (required)",
  "status": "string (required: pending|in_progress|completed|blocked|skipped)",
  "reason": "string (optional, for blocked/skipped)"
}
```

---

### phase_add_checkpoint

Add a checkpoint note to a phase.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "phase_order": "integer (required)",
  "note": "string (required)",
  "context": "object (optional)"
}
```

## Session Tools

### session_start

Start a new agent session.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (optional)",
  "agent_type": "string (required: opus|sonnet|haiku)",
  "context": "object (optional)"
}
```

**Output:**
```json
{
  "success": true,
  "session": {
    "session_id": "ses_abc123xyz",
    "agent_type": "opus",
    "plan": "my-plan",
    "status": "active"
  }
}
```

---

### session_end

End a session with status and summary.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)",
  "status": "string (required: completed|failed)",
  "summary": "string (optional)"
}
```

---

### session_log

Add a work log entry to an active session.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)",
  "message": "string (required)",
  "type": "string (optional: info|warning|error|success|checkpoint)",
  "data": "object (optional)"
}
```

---

### session_handoff

Prepare session for handoff to another agent.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)",
  "summary": "string (required)",
  "next_steps": ["string"],
  "blockers": ["string"],
  "context_for_next": "object (optional)"
}
```

---

### session_resume

Resume a paused session.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "session": {...},
  "handoff_context": {
    "summary": "Previous work summary",
    "next_steps": ["Continue with..."],
    "blockers": []
  }
}
```

---

### session_replay

Get replay context for a session.

**Scopes:** `read`

**Input:**
```json
{
  "session_id": "string (required)"
}
```

**Output:**
```json
{
  "success": true,
  "replay_context": {
    "session_id": "ses_abc123",
    "progress_summary": {...},
    "last_checkpoint": {...},
    "decisions": [...],
    "errors": [...]
  }
}
```

---

### session_continue

Create a new session that continues from a previous one.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)",
  "agent_type": "string (optional)"
}
```

---

### session_artifact

Add an artifact (file) to a session.

**Scopes:** `write`

**Input:**
```json
{
  "session_id": "string (required)",
  "path": "string (required)",
  "action": "string (required: created|modified|deleted)",
  "metadata": "object (optional)"
}
```

---

### session_list

List sessions with optional filtering.

**Scopes:** `read`

**Input:**
```json
{
  "plan_slug": "string (optional)",
  "status": "string (optional)",
  "limit": "integer (optional)"
}
```

## State Tools

### state_set

Set a workspace state value.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "key": "string (required)",
  "value": "any (required)",
  "category": "string (optional)"
}
```

---

### state_get

Get a workspace state value.

**Scopes:** `read`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "key": "string (required)"
}
```

---

### state_list

List all state for a plan.

**Scopes:** `read`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "category": "string (optional)"
}
```

## Task Tools

### task_update

Update a task within a phase.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "phase_order": "integer (required)",
  "task_identifier": "string|integer (required)",
  "status": "string (optional: pending|completed)",
  "notes": "string (optional)"
}
```

---

### task_toggle

Toggle a task's completion status.

**Scopes:** `write`

**Input:**
```json
{
  "plan_slug": "string (required)",
  "phase_order": "integer (required)",
  "task_identifier": "string|integer (required)"
}
```

## Template Tools

### template_list

List available plan templates.

**Scopes:** `read`

**Output:**
```json
{
  "success": true,
  "templates": [
    {
      "slug": "feature-development",
      "name": "Feature Development",
      "description": "Standard feature workflow",
      "phases_count": 5,
      "variables": [
        {
          "name": "FEATURE_NAME",
          "required": true
        }
      ]
    }
  ]
}
```

---

### template_preview

Preview a template with variable substitution.

**Scopes:** `read`

**Input:**
```json
{
  "slug": "string (required)",
  "variables": {
    "FEATURE_NAME": "Authentication"
  }
}
```

---

### template_create_plan

Create a plan from a template.

**Scopes:** `write`

**Input:**
```json
{
  "template_slug": "string (required)",
  "variables": "object (required)",
  "title": "string (optional, overrides template)",
  "activate": "boolean (optional, default false)"
}
```

## Content Tools

### content_generate

Generate content using AI.

**Scopes:** `write`

**Input:**
```json
{
  "prompt": "string (required)",
  "provider": "string (optional: claude|gemini|openai)",
  "config": {
    "temperature": 0.7,
    "max_tokens": 4000
  }
}
```

---

### content_batch_generate

Generate content for a batch specification.

**Scopes:** `write`

**Input:**
```json
{
  "batch_id": "string (required)",
  "provider": "string (optional)",
  "dry_run": "boolean (optional)"
}
```

---

### content_brief_create

Create a content brief for later generation.

**Scopes:** `write`

---

### content_brief_get

Get a content brief.

**Scopes:** `read`

---

### content_brief_list

List content briefs.

**Scopes:** `read`

---

### content_status

Get batch generation status.

**Scopes:** `read`

---

### content_usage_stats

Get AI usage statistics.

**Scopes:** `read`

---

### content_from_plan

Generate content based on plan context.

**Scopes:** `write`

## Error Responses

All tools return errors in this format:

```json
{
  "error": "Error message",
  "code": "error_code"
}
```

Common error codes:
- `validation_error` - Invalid input
- `not_found` - Resource not found
- `permission_denied` - Insufficient permissions
- `rate_limited` - Rate limit exceeded
- `service_unavailable` - Circuit breaker open

## Circuit Breaker

Tools use circuit breaker protection for database calls. When the circuit opens:

```json
{
  "error": "Agentic service temporarily unavailable",
  "code": "service_unavailable"
}
```

The circuit resets after successful health checks.
