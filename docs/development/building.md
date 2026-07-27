<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Building

## Go workspace

The module is `dappco.re/go/agent`, rooted at `go/`. It participates in a Go workspace
(`go.work`) that resolves all `dappco.re/go/*` dependencies locally via the submodules
under `external/`. Run Go tooling from `go/`:

- Development / default: `cd go && go build ./...`, `cd go && go test ./...`
- CI / reproducibility: add `GOWORK=off` (and optionally `GOFLAGS=-mod=mod`) when running
  `go test`, `go vet`, and `go mod tidy` from `go/`.

## PHP dependencies

```bash
composer install
```

The Composer package is `lthn/agent`. It depends on `lthn/php` (the foundation framework)
at runtime, and on `orchestra/testbench`, `pestphp/pest`, and `livewire/livewire` for
development.

## The binary

A single binary builds from `go/cmd/core-agent`:

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
