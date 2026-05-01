<!-- SPDX-License-Identifier: EUPL-1.2 -->

# core/agent

> AI agent orchestration platform for the Core ecosystem — dispatch, verify, sync, fleet — with first-class Claude / Codex / Hermes / Google provider integrations.

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

## What it is

`core-agent` is a single Go binary that runs as an MCP server (stdio for
Claude Code integration, HTTP for cross-agent communication) plus a CLI
that dispatches work across multiple AI providers. It owns:

- **Dispatch** — fan out a Mantis ticket to a sandboxed worker
  (Claude / Codex / Hermes / Google) running in `.core/workspace/`.
- **Fleet sync** — pull / merge / push across the Core ecosystem repos
  per `agents.yaml`.
- **OpenBrain** — durable memory + cross-agent messaging via
  Postgres + Qdrant + Ollama (homelab stack).
- **Provider integrations** — per-provider plugin trees under
  `provider/{claude,codex,hermes,google}/`. Claude's tree is the
  most fleshed out (8 plugins shipped via Claude Code marketplace).

## Repository Layout

```
agent/
├── go/                          Go module — module path: dappco.re/go/agent
│   ├── cmd/core-agent/          Binary entry point (mcp + serve)
│   ├── pkg/agentic/             Dispatch, verify, remote, mirror, queue
│   ├── pkg/brain/               OpenBrain client (recall + remember)
│   ├── pkg/monitor/             Background monitor + repo sync
│   ├── pkg/lib/                 Workspace extraction + flow templates
│   ├── pkg/runner/              Local + container runners
│   └── pkg/prompts/             Embedded persona + flow templates
├── php/                         PHP package — Laravel module + Boot, Actions,
│                                Agentic for the lthn.ai hosted service
├── provider/
│   ├── claude/                  Claude Code plugin sources (marketplace at
│   │   ├── core/                .claude-plugin/marketplace.json)
│   │   ├── core-go/             — 8 plugins: core, core-go, core-php,
│   │   ├── core-php/            devops, infra, research, hermes_runner_mcp,
│   │   ├── devops/              camofox_mcp
│   │   ├── infra/
│   │   ├── research/
│   │   ├── hermes_runner_mcp/
│   │   ├── camofox_mcp/
│   │   └── plugins/             marketplace-flavoured subset
│   ├── codex/                   codex-cli configs + harness
│   ├── hermes/                  Hermes plugin sources + skills
│   └── google/                  Google Gemini integration scaffolding
├── vm/docker/                   Containerised dev stack (Dockerfile + compose)
├── .core/                       Runtime workspace seed (agents.yaml + workspace.yaml)
├── docs/                        RFCs, onboarding, audits
├── go.work + external/          Dev workspace mode (see CLAUDE.md)
├── Taskfile.yaml                Build orchestration (module-graph refresh, etc.)
└── module-graph.json            Authoritative dappco.re/go/* dep snapshot
```

## Quickstart

### As a Go module

```bash
go get dappco.re/go/agent@latest
```

### As a binary

```bash
cd go
go install ./cmd/core-agent/
core-agent mcp        # MCP stdio mode for Claude Code
core-agent serve      # HTTP daemon mode
```

### As a Claude Code plugin marketplace

```bash
claude plugin marketplace add https://github.com/dappcore/agent
claude plugin install core-agent
```

The provider/claude/* tree is the source for the plugins. The
`.claude-plugin/marketplace.json` at repo root publishes them.

## Build + Test

```bash
cd go
GOWORK=off go test -count=1 ./...
GOWORK=off go vet ./...
golangci-lint run --timeout=5m --tests=false ./...
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

Cross-compile for Charon (homelab Linux box):

```bash
cd go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o core-agent-linux ./cmd/core-agent/
```

`module-graph.json` at repo root is the authoritative `dappco.re/go/*` dep
snapshot — regenerate via `task module-graph:refresh` after dep bumps.

## Configuration

Runtime config lives at `.core/agents.yaml` (also extracted into
`~/Code/.core/agents.yaml` on first run via `core-agent setup`). Tunes
dispatch concurrency, default agent type, container runtime, image,
GPU passthrough, and per-provider quotas.

The repo seeds workspace template at `.core/workspace.yaml` —
extracted into per-task workspaces under `~/Code/.core/workspace/<task>/`.

## CI

- **Internal** (homelab, full sonar.lthn.sh detail): Woodpecker pipeline
  defined in `.woodpecker.yml`.
- **Public** (badges + mirror analytics): GitHub Actions workflow at
  `.github/workflows/ci.yml` — pushes coverage to Codecov + analysis
  to SonarCloud.

## Branch Model

- `dev` — active development. All Cladius / codex lane work lands here
  first.
- `main` — squash-stable. Promoted via the squash-and-push gate on the
  public mirror only.

## Licence

EUPL-1.2 — see [LICENCE](LICENCE).

## Authorship

Maintained by Cladius (Snider's in-house Opus persona) via the
`agent/cladius` workspace at `forge.lthn.sh/agent/cladius`. Most
substantive commits land via the codex lane pattern documented in
`factory/`. Per-provider integration owners:

- claude/  Cladius
- codex/   Codex (cloud) + Cyclops (lane runner persona)
- hermes/  Hermes (homelab agent at chat.lthn.sh)
- google/  reserved for Gemini integration
