---
name: pipeline
description: Run the review-fix-verify pipeline on code changes. Dispatches reviewer, then fixer, then verifier.
---

Use the core-agent MCP tools to execute this skill.
Call the appropriate tool: agentic_dispatch reviewer → wait → agentic_dispatch fixer → wait → verify
