<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Building

## Go module

The module is `dappco.re/go/agent`, rooted at `go/`. Every `dappco.re/go/*` dependency
resolves from its published module tag — there is no workspace and no `replace`
directive, so a checkout builds the same way everywhere. Run Go tooling from `go/`:

- Development / default: `cd go && go build ./...`, `cd go && go test ./...`
- CI / reproducibility: add `GOWORK=off` (and optionally `GOFLAGS=-mod=mod`) when running
  `go test`, `go vet`, and `go mod tidy` from `go/`. `GOWORK=off` proves resolution comes
  from the tags alone, even if a stray `go.work` exists higher up the tree.

## PHP dependencies

```bash
composer install
```

The Composer package is `lthn/agent`. It depends on `lthn/php` (the foundation framework)
at runtime, and on `orchestra/testbench`, `pestphp/pest`, and `livewire/livewire` for
development.

## The binary

`task` is the shortest path, from the repo root:

```bash
task build         # → bin/core-agent (and prints its version to prove it runs)
task build:lthn    # → bin/lthn-agent, the crew alias lthn/desktop stages
task check         # build + vet + test, the CI gates
task cov           # coverage → go/coverage.out, prints the total
task --list        # everything available
```

Every Go task runs with `GOWORK=off`, deliberately: the module must build from
`go.mod` alone, because that is what CI does and what a consumer gets. The
sibling `dappco.re/*` modules are dependencies, never a local checkout — see
the `no-vendored-ecosystem` CI job.

Directly, from `go/`:

```bash
cd go
go build ./cmd/core-agent/        # build core-agent
go install ./cmd/core-agent/      # install to $GOPATH/bin
go build ./...                    # build all packages
```

The same source ships under two names — `core-agent` and `lthn-agent`. Build the
family-consistent name by setting the output, and the binary detects its name from
`argv[0]`:

```bash
go build -o lthn-agent ./cmd/core-agent/
```

## MCP + serve modes

The binary *is* the MCP server. The `mcp` (stdio) and `serve` (HTTP) commands are
registered by the shared `dappco.re/go/mcp` service the binary mounts:

```bash
core-agent mcp        # MCP server over stdio — what an IDE connects to
core-agent serve      # HTTP MCP daemon — cross-agent communication
```

The tool surface (dispatch, plans, brain, messaging, `lemma_send`, …) is registered by the
`agentic`, `brain`, and `lemma` subsystems into that one service — there are no separate
per-server binaries.
