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

Use the `mcp__core__brain_recall` tool with:
- query: $ARGUMENTS.query
- top_k: 5
- filter with project and type if provided

Show results with score, type, project, date, and content preview.
