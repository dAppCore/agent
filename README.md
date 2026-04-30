<!-- SPDX-License-Identifier: EUPL-1.2 -->

# core-agent

> Agent orchestrator — codex/claude/google fan-out, dispatch, fleet, sync, QA

[![CI](https://github.com/dappcore/agent/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/dappcore/agent/actions/workflows/ci.yml)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=alert_status)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Coverage](https://codecov.io/gh/dappcore/agent/branch/dev/graph/badge.svg)](https://codecov.io/gh/dappcore/agent)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=security_rating)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=code_smells)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=dappcore_agent&metric=ncloc)](https://sonarcloud.io/dashboard?id=dappcore_agent)
[![Go Reference](https://pkg.go.dev/badge/dappco.re/go/agent.svg)](https://pkg.go.dev/dappco.re/go/agent)
[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](https://eupl.eu/1.2/en/)


A monorepo of [Claude Code](https://claude.ai/code) plugins for the Host UK federated monorepo.

## Plugins

| Plugin | Description | Commands |
|--------|-------------|----------|
| **[code](./claude/code)** | Core development - hooks, scripts, data collection | `/code:remember`, `/code:yes` |
| **[review](./claude/review)** | Code review automation | `/review:review`, `/review:security`, `/review:pr` |
| **[verify](./claude/verify)** | Work verification before commit/push | `/verify:verify`, `/verify:ready` |
| **[qa](./claude/qa)** | Quality assurance fix loops | `/qa:qa`, `/qa:fix`, `/qa:check` |
| **[ci](./claude/ci)** | CI/CD integration | `/ci:ci`, `/ci:workflow`, `/ci:fix` |

## Installation

```bash
# Install all plugins via marketplace
claude plugin add host-uk/core-agent

# Or install individual plugins
claude plugin add host-uk/core-agent/claude/code
claude plugin add host-uk/core-agent/claude/review
claude plugin add host-uk/core-agent/claude/qa
```

## Quick Start

```bash
# Code review staged changes
/review:review

# Run QA and fix all issues
/qa:qa

# Verify work is ready to commit
/verify:verify

# Check CI status
/ci:ci
```

## Core CLI Integration

These plugins enforce the `core` CLI for development commands:

| Instead of... | Use... |
|---------------|--------|
| `go test` | `core go test` |
| `go build` | `core build` |
| `golangci-lint` | `core go lint` |
| `composer test` | `core php test` |
| `./vendor/bin/pint` | `core php fmt` |

## Plugin Details

### code

The core plugin with hooks and data collection skills:

- **Hooks**: Auto-format, debug detection, dangerous command blocking
- **Skills**: Data collection for archiving OSS projects (whitepapers, forums, market data)
- **Commands**: `/code:remember` (persist facts), `/code:yes` (auto-approve mode)

### review

Code review automation:

- `/review:review` - Review staged changes or commit range
- `/review:security` - Security-focused review
- `/review:pr [number]` - Review a pull request

### verify

Work verification:

- `/verify:verify` - Full verification (tests, lint, format, debug check)
- `/verify:ready` - Quick check if ready to commit

### qa

Quality assurance:

- `/qa:qa` - Run QA pipeline, fix all issues iteratively
- `/qa:fix <issue>` - Fix a specific issue
- `/qa:check` - Check without fixing

### ci

CI/CD integration:

- `/ci:ci` - Check CI status
- `/ci:workflow <type>` - Generate GitHub Actions workflow
- `/ci:fix` - Analyse and fix failing CI

## Development

### Adding a new plugin

1. Create `claude/<name>/.claude-plugin/plugin.json`
2. Add commands to `claude/<name>/commands/`
3. Add hooks to `claude/<name>/hooks.json` (optional)
4. Register in `.claude-plugin/marketplace.json`

### Testing locally

```bash
claude plugin add /path/to/core-agent
```

## License

EUPL-1.2

## Links

- [Host UK](https://host.uk.com)
- [Claude Code Documentation](https://docs.anthropic.com/claude-code)
- [Issues](https://github.com/host-uk/core-agent/issues)
