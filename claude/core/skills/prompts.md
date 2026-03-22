---
name: prompts
description: Browse and read from the prompts library — personas, tasks, flows, templates
arguments:
  - name: action
    description: list or read
    required: true
  - name: type
    description: persona, task, flow, prompt
  - name: slug
    description: The slug to read (e.g. secops/developer, code/review, go)
---

# Prompts Library

Access the embedded prompts at `~/Code/core/agent/pkg/prompts/lib/`.

## List

```bash
# List all of a type
find ~/Code/core/agent/pkg/prompts/lib/$ARGUMENTS.type -name "*.md" -o -name "*.yaml" | sed "s|.*/lib/$ARGUMENTS.type/||;s|\.\(md\|yaml\)$||" | sort
```

## Read

```bash
# Read a specific prompt
cat ~/Code/core/agent/pkg/prompts/lib/$ARGUMENTS.type/$ARGUMENTS.slug.md 2>/dev/null || \
cat ~/Code/core/agent/pkg/prompts/lib/$ARGUMENTS.type/$ARGUMENTS.slug.yaml 2>/dev/null || \
echo "Not found: $ARGUMENTS.type/$ARGUMENTS.slug"
```

## Quick Reference

| Type | Path | Examples |
|------|------|----------|
| persona | `lib/persona/` | `secops/developer`, `code/backend-architect`, `smm/tiktok-strategist` |
| task | `lib/task/` | `bug-fix`, `code/review`, `code/refactor`, `new-feature` |
| flow | `lib/flow/` | `go`, `php`, `ts`, `docker`, `release` |
| prompt | `lib/prompt/` | `coding`, `verify`, `conventions`, `security` |
