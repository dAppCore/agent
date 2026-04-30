---
name: recall
description: Search OpenBrain for memories and context
arguments:
  - name: query
    description: What to search for
    required: true
  - name: project
    description: Filter by project
  - name: type
    description: Filter by type (decision, plan, convention, architecture, observation, fact)
---

Use the core-agent MCP tool `brain_recall` with:
- query: `$ARGUMENTS.query`
- top_k: `5`
- filter.project: `$ARGUMENTS.project` if provided
- filter.type: `$ARGUMENTS.type` if provided

Show results with score, type, project, date, and a short content preview.
