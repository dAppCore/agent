---
name: content
description: Use CoreAgent content generation, briefs, batch status, usage stats, SEO schema, and Natural Progression SEO scheduling
args: "[generate|batch|brief|status|usage|from-plan|schema|seo-schedule] [options]"
---

# Content Workflows

Use this family for platform-backed content generation and SEO support.

## Registered CLI Commands

```bash
core-agent generate --prompt="Draft a release note" --provider=claude
core-agent content schema generate --type=howto --title="Set up the workspace" --steps='[...]'
```

## Action Or MCP Surface

When Core actions or MCP wrappers are available, route these feature requests to the matching action instead of inventing shell commands:

| Feature | Core action |
|---------|-------------|
| Batch generation | `content.batch.generate` |
| Brief create/get/list | `content.brief.create`, `content.brief.get`, `content.brief.list` |
| Batch status | `content.status` |
| Usage statistics | `content.usage.stats` |
| Plan-derived content | `content.from.plan` |
| SEO schema | `content.schema.generate` |
| Natural Progression SEO scheduling | `content_seo_schedule` MCP tool |

## SEO Scheduling

When the MCP tool is available, use `content_seo_schedule` to create a pending Natural Progression SEO revision:

```json
{
  "page_id": "/help/hosting",
  "content": "Updated copy"
}
```

Googlebot-triggered scheduling is handled by CoreAgent middleware; do not publish scheduled revisions directly unless the user explicitly asks.
