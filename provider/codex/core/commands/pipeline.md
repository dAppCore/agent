---
name: pipeline
description: Run the multi-stage review pipeline on code changes
args: [commit-range|--pr=N|--stage=NAME|--skip=fix]
---

# Review Pipeline

Run a staged code review pipeline using specialised roles for security, fixes, tests, architecture, and final verification.

## Usage

```
/core:pipeline
/core:pipeline HEAD~3..HEAD
/core:pipeline --pr=123
/core:pipeline --stage=security
/core:pipeline --skip=fix
```

## Pipeline Stages

| Stage | Role | Purpose | Modifies Code? |
|------|------|---------|----------------|
| 1 | Security Engineer | Threat analysis, injection, tenant isolation | No |
| 2 | Senior Developer | Fix critical findings from Stage 1 | Yes |
| 3 | API Tester | Run tests and identify coverage gaps | No |
| 4 | Backend Architect | Check architecture fit and conventions | No |
| 5 | Reality Checker | Evidence-based final verdict | No |

## Process

1. Gather the diff and changed file list for the requested range
2. Identify the affected package so tests can run in the right place
3. Dispatch each stage with `agentic_dispatch`, carrying forward findings from earlier stages
4. Aggregate the outputs into a single report with verdict and required follow-up

## Single Stage Mode

When `--stage=NAME` is passed, run only one stage:

| Name | Stage |
|------|-------|
| `security` | Stage 1 |
| `fix` | Stage 2 |
| `test` | Stage 3 |
| `architecture` | Stage 4 |
| `reality` | Stage 5 |
