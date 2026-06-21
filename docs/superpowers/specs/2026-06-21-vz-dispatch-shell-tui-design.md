<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Design — VZ-first containerised dispatch + container shell TUI for core-agent

**Date:** 2026-06-21
**Status:** Design approved (decomposition + ordering); awaiting spec review → implementation plan
**Author:** Cladius (brainstorming session with Snider)
**Drives:** `core/agent` ⟶ consumes `core/go-container` (`dappco.re/go/container`)

---

## 1. Problem & Intent

core-agent dispatches coding agents (codex/claude/gemini) inside containers. Today the
container execution path is **string-based CLI shelling**: `resolveContainerRuntime`
picks a runtime name by probing `$PATH`, and `containerCommandFor` builds a
`docker|container|podman run --rm -v …` argv that is then spawned as a host process.
`dappco.re/go/container` is **not a dependency**, and there is **no VZ path**.

Two intents drive this work:

1. **Run agent dispatch on Apple Virtualization.framework directly (VZ), in place of
   Docker, when available** — daemon-free, hardware-isolated, App-Sandbox-compatible,
   via go-container's already-built `VZProvider` (in-process `tmc/apple` purego
   bindings — "direct calls to the Apple OS API").
2. **Add a `core-agent shell <id>` route that drops the user into an interactive shell
   inside a running container/VM.**

### 1.1 Key prior-art finding

core-agent's **own** `RFC.md §15.5.3 (Apple Container Dispatch)** already specifies the
go-container integration this work needs — `container.detect`, `container.run`,
`build.linuxkit.resolve` actions; LinuxKit immutable images (`core-dev`/`core-ml`/
`core-minimal`); apple→docker fallback; `WithGPU` Metal passthrough. **The code never
implemented it** — detection was re-built as `$PATH` probes and dispatch shells out to
CLIs directly. So most of this work is *closing the existing spec↔code gap in §15.5.3*,
then adding **VZ on top** as the top-priority runtime (per `RFC.vz.md`), plus the shell
TUI.

### 1.2 Decisions locked in brainstorming

| Decision | Choice | Consequence |
|----------|--------|-------------|
| Scope | **Full dispatch-in-VZ replacement** | Needs the LinuxKit agent-guest-image pipeline (SP3), not just plumbing |
| Integration | **Import `dappco.re/go/container` directly** | In-process `VZProvider.Run/Exec`; `tmc/apple`+`purego` enter core-agent's dep tree → §2.1 supply-chain gate + signing land on the core-agent binary |
| Signing/entitlement | **Signed entitled build + auto-fallback** | VZ is "best available", never a hard requirement; dev/CI/Linux fall back apple→docker |
| Shell TUI shape | **`core-agent shell <id>` raw PTY in current terminal** | OCI: `exec -it`; VZ: needs a NEW interactive vsock protocol (vzproto is batch-only today) |

---

## 2. Architecture

### 2.1 The dispatch fork

```
                       resolved runtime
                              │
        ┌─────────────────────┴──────────────────────┐
        │ OCI-CLI path (EXISTING, unchanged)          │  in-process VZ path (NEW)
        │ docker | apple(container) | podman          │  vz
        │ containerCommandFor → "run --rm -v …" argv  │  container.NewVZProvider().Run(image, opts)
        │ spawned as a host process (PID tracked)     │  VZProvider lifecycle in-process
        └─────────────────────────────────────────────┘  registry: ~/.core/containers.json (shared)
```

The OCI runtimes (docker/apple/podman) genuinely share the `run --rm -v` argv surface,
so they collapse to a binary-name swap over one `containerCommandFor`. **VZ is a
different execution model** — in-process Go booting LinuxKit kernel+initrd+disk, not an
OCI image — so the dispatch path forks rather than adding an enum arm.

### 2.2 Detection & resolution

Replace the `$PATH`-probing `resolveContainerRuntime` with go-container's detection:

- `container.Detect()` → highest-priority `ContainerRuntime`. **Verified:** `DetectAll()`
  already runs `detectApple → detectVZ → detectDocker → detectPodman → detectLinuxKit`,
  so **VZ is already surfaced by go-container detection — no go-container change needed
  for SP1's detection consumption.**
- **Dispatch branches on the runtime string, NOT on a polymorphic provider.** `Verified:`
  the `Provider` interface is `Build/Run/Encrypt/Decrypt` only — `Stop/Kill/Exec/Logs/
  Wait/Remove/Tracked` are concrete methods on `*VZProvider`/`*AppleProvider`, not on the
  interface. So `container.ProviderFor(rt)` cannot run a lifecycle. The dispatch fork
  therefore routes **vz → concrete `container.NewVZProvider()`** (which has Exec/Stop/…)
  and **OCI → existing argv** — it does not try to unify them behind `Provider`.
- `agents.yaml` `dispatch.runtime` gains `vz` to the existing `auto|apple|docker|podman`.
- `CORE_AGENT_RUNTIME` env override still wins (tests/CI).

VZ selection additionally requires the binary to be **signed + entitled** (§2.4).
`container.IsVZAvailable()` reports framework-load + arch; the *entitlement* cannot be
cheaply probed (RFC.vz.md §2.2) — an unentitled caller sees `Available()==true` and
receives the framework's verbatim entitlement error at `Run`. Therefore core-agent
treats a VZ `Run` entitlement error as a **fallback trigger**, not a hard failure.

### 2.3 Auto-fallback contract

`auto` resolves to the first *usable* runtime. "Usable" for VZ means: arch ok AND
(entitled OR `CONTAINER_VZ_LIVE` opt-in). On a VZ `Run` failure whose error names the
missing entitlement, dispatch retries down the chain (apple→docker) and records the
downgrade in the workspace status. A plain `go build` / CI run therefore never blocks on
VZ — it silently uses apple/docker.

### 2.4 Build & signing

The VZ path only boots from a binary carrying `com.apple.security.virtualization`.
Release builds are codesigned with the entitlement (operator-owned, SP0). Dev/CI builds
are unsigned and fall back. This is documented as a build-pipeline dependency, not
implemented in Go.

---

## 3. Sub-projects

Each sub-project is independently shippable and testable. Order:
**SP0 (parallel) → SP1 → SP2 → SP3 → SP4 → SP5.**

### SP0 — Operator gates (non-code, parallel; blocks merge not dev)

- **(a) Supply-chain review** of `github.com/tmc/apple` (`virtualization` + `x/vzkit`
  only — never `private/*`) and `ebitengine/purego`, per RFC.vz.md §2.1. Pin exact
  versions; vendoring acceptable. **Required before VZ deps merge to the default branch.**
- **(b) Code-signing + entitlement provisioning** — `com.apple.security.virtualization`
  on core-agent release builds; signing identity in the release pipeline.

**Done when:** review sign-off recorded; a signed entitled core-agent boots a VZ VM on
an Apple-silicon host.

### SP1 — go-container dependency + detection seam (foundation, no behaviour change)

- Add `dappco.re/go/container` to `go/go.mod` + `go.work` wiring.
- New seam (e.g. `pkg/agentic/runtime_container.go` or a small `pkg/containerrt`)
  wrapping `container.Detect()/DetectAll()/ProviderFor()`.
- Replace `resolveContainerRuntime`/`runtimeAvailable`/`containerRuntimeBinary`
  internals with go-container detection; **keep the same `string` return + existing OCI
  argv path** so docker/apple/podman behaviour is byte-for-byte unchanged.
- Add `vz` to the runtime enum, `agents.yaml` schema, and `DispatchConfig`.
- **Supply-chain gate timing (corrected — see R4):** go-container's `Detect()` lives in
  the same `package container` as the darwin-only `vz.go`, which imports `tmc/apple`. So
  importing `container` *for detection alone* transitively compiles `tmc/apple` **on
  darwin** — there is no build-tag that keeps it out of a darwin build. Therefore **SP0(a)
  is on SP1's darwin critical path** (SP1 must not merge to a release branch before
  sign-off). A `//go:build vz` tag (NOT a cgo tag — VZ via purego is no-cgo) gates only
  core-agent's *own* VZ-dispatch code (SP2), not the transitive dependency. Non-darwin
  builds resolve `vz_other.go` and stay `tmc/apple`-free.

**Done when:** detection routes through go-container; `vz` is a recognised
(but not-yet-bootable) runtime; all existing dispatch tests pass unchanged.

### SP2 — VZ in-process dispatch fork

- Fork `spawnAgent`/`containerCommandFor` call-site (`dispatch.go:~712`): when resolved
  runtime is `vz`, call `container.NewVZProvider().Run(image, opts…)` in-process instead
  of building an argv.
- Map dispatch config → `RunOption`s: `WithMemory`, `WithCPUs`, `WithVolumes`
  (workspace + meta), `WithEnv` (keys via SP3 injection), `WithName`.
- Track the VM in the **shared** `~/.core/containers.json` registry and stream the serial
  console to `~/.core/logs/{id}.log` (go-container already owns both conventions).
- Agent command execution inside the VM uses `VZProvider.Exec(id, cmd, args…)` (batch).
- Auto-fallback per §2.3 on entitlement error.
- **Tests:** configuration-construction tests run anywhere; live-boot gated on
  `CONTAINER_VZ_LIVE=1` + signed/entitled binary.

**Done when:** on a signed/entitled host, `dispatch.runtime: vz` boots a minimal VM,
runs a command via the agent, and lands status/logs in the shared registry; unentitled
hosts fall back cleanly.

### SP3 — LinuxKit agent-guest-image pipeline (heavy; own spec)

The blocker for "dispatch *every* agent in VZ". VZ cannot run the OCI `core-dev` image
— it needs the RFC.vz.md §4 guest artefact set (`kernel`, `initrd.img`, `cmdline`,
`disk.img`, 512-byte sector-aligned).

- **LinuxKit YAML** producing kernel+initrd with: agent toolchains (node/go/python), the
  agent CLIs (codex/claude/gemini), `vzagent` baked in as a service,
  `CONFIG_VIRTIO_VSOCKETS=y`, agent service `CAP_SYS_BOOT`.
- **Workspace delivery — virtio-fs (decided, not open).** The dispatch model REQUIRES a
  **host-visible read-write workspace** — agents commit to the host repo and push, so the
  workspace cannot live inside a disk image. **Verified:** go-container's VZProvider wires
  **block devices only** (`vzAttachStorage` → `VZVirtioBlockDeviceConfiguration`), but the
  upstream binding `tmc/apple v0.6.12` **does** expose directory sharing
  (`VZVirtioFileSystemDeviceConfiguration`, `NewVirtioFileSystemDeviceConfigurationWith
  Tag`, `VZSingleDirectoryShare`) and `x/vzkit` ships a `virtiofs` subpackage. So SP3
  includes a **go-container-side change**: add a virtio-fs directory-share device to
  VZProvider (host workspace dir, tagged), and the guest mounts the tag. This also
  **extends RFC.vz.md §4** (the guest contract currently lists block devices only) — that
  RFC needs a virtio-fs workspace clause. Raw block disk remains the mechanism for the
  immutable rootfs; virtio-fs is the writable workspace.
- **Spec baking** (~/spec/ read-only) per core-agent RFC §15.5.2.
- **Secret injection over vsock** — `OPENAI_API_KEY`/`ANTHROPIC_API_KEY`/`GEMINI_API_KEY`
  + git identity delivered to the guest over the control channel (NOT kernel cmdline,
  NOT `ps`-visible), mirroring the OCI path's `-e KEY` passthrough.
- **`build.linuxkit.resolve` action** (RFC §15.5.3) — resolve `core-dev`/`core-ml`/
  `core-minimal` → cached bootable artefact set; integrate go-build's LinuxKit builder.
- **Tests:** image-build smoke (CI artefact presence) + a live boot-and-exec on an
  entitled host.

**Done when:** `build.linuxkit.resolve("core-dev")` yields a bootable VZ artefact set
whose guest runs codex/claude/gemini against a mounted workspace with injected keys.

### SP4 — Interactive shell: vsock PTY protocol + `core-agent shell <id>`

`vzproto` today is **batch-only** (one `Request`→one buffered `Response`; `vzagent`
captures stdout/stderr via `capWriter` and `cmd.Run()`). An interactive shell needs
streaming + a PTY. This is a **go-container change** plus a core-agent CLI.

- **go-container — vzproto interactive mode:** add a framed channel for an interactive
  session: `open(pty, cols, rows)`, bidirectional `stdin`/`stdout` data frames,
  `resize(cols, rows)`, `exit(code)`. Keep the batch protocol intact alongside it;
  bump a protocol version. Unit-test fully over `net.Pipe` (no VM).
- **go-container — vzagent PTY:** allocate a PTY (e.g. `creack/pty` or raw `syscall`),
  spawn the shell attached to it, pump both directions, honour resize and exit. Reship
  the static guest binary; SP3's image must bake the new `vzagent`.
- **core-agent — `core-agent shell <id>`:** new CLI subcommand. Put the local terminal in
  raw mode; for VZ, dial the control vsock, send `open`, multiplex `os.Stdin`↔stdout over
  the interactive frames, forward `SIGWINCH`→`resize`, restore the terminal on exit. For
  docker/podman, exec `<rt> exec -it <id> $SHELL`; for apple, **reuse the existing
  `AppleProvider.ExecInteractive(id, cmd...)`** rather than hand-rolling `container exec
  -it`. Reuse the `tui.go` quoting helpers for argv safety. Optionally expose a hub
  `/container/:id/shell` route later (out of scope for this SP).
- **Tests:** protocol `_Good/_Bad/_Ugly` over `net.Pipe`; OCI `exec -it` argv test;
  raw-mode/restore unit isolation.

**Done when:** `core-agent shell <id>` gives a working interactive shell into a running
OCI container AND a running VZ VM, with working resize and clean exit.

### SP5 — Specced-but-incomplete cleanup

- **Metal GPU passthrough** — wire `WithGPU` through the VZ path (RFC.vz.md §15, RFC
  §15.5.3); no-op until Apple's framework exposes it, but the option + capability
  (`ContainerRuntime.HasGPU`) plumb end-to-end.
- **go-container GOAL-STATUS "Remaining for separate passes":** macOS 26+ CLI flag
  verification (GPU flag, JSON schema, digest format); AX polish audit; RFC §3.3 AMI/GCP
  formats; v0.9.0 audit findings (legacy-log-package, ax7-triplet-gaps, example-gaps);
  RFC cross-reference link resolution.

**Done when:** the gap inventory (§4) items are each either closed or explicitly
deferred with a recorded reason.

---

## 4. Gap inventory — "specced but not completed"

Grounded in the RFCs + GOAL files, not guessed.

**core-agent RFC §15.5.3 vs `pkg/agentic/dispatch.go`:**
- go-container not imported; `container.detect` / `container.run` /
  `build.linuxkit.resolve` actions absent — detection is `$PATH` probes.
- LinuxKit immutable-image pipeline not wired (uses raw `core-dev` image name).
- Spec-baking (~/spec/ read-only, §15.5.2) missing on the OCI path.
- VZ runtime entirely absent from core-agent.

**RFC.vz.md (go-container — built but gated/incomplete):**
- §2.1 `tmc/apple` supply-chain review not cleared.
- §2.2 signed/entitled binary not provisioned.
- §8 live-boot tests gated (need entitled signed test binary).
- §15 Metal GPU passthrough pending Apple framework.
- **Interactive PTY exec not specced/built** (batch-only) — the shell-TUI blocker.
- **No virtio-fs directory sharing** — VZProvider wires block devices only, so the
  workspace can't be host-visible read-write; `tmc/apple v0.6.12` + `x/vzkit/virtiofs`
  expose it but go-container doesn't use it. RFC.vz.md §4 (guest contract) lists block
  devices only and needs a virtio-fs workspace clause.

**go-container GOAL-STATUS.md "Remaining":**
- macOS 26+ CLI-flag verification; AX polish audit; RFC §3.3 AMI/GCP formats; v0.9.0
  audit findings; RFC cross-reference resolution.

### 4.1 go-container-side work this introduces

"Import go-container directly" is mostly *consuming* it, but three SPs require changes
**inside go-container** (so SP0's supply-chain review scope and the per-SP specs cover the
right surface):

- **SP1 — none for detection** (`Detect()` already includes VZ). Possibly a thin
  string/`ContainerRuntime` accessor.
- **SP3 — virtio-fs device** on VZProvider (workspace directory share) + a guest mount;
  **+ RFC.vz.md §4 update**.
- **SP4 — vzproto interactive/PTY mode + vzagent PTY rewrite** + reshipped guest binary;
  **+ RFC.vz.md §5 update**.

SP2 consumes the concrete `*VZProvider` lifecycle (Run/Exec/Stop/Logs/Wait) as-is.

---

## 5. Cross-cutting conventions

- **Errors:** `core.E("pkg.Method", "message", err)` / `core.Result{Value, OK}` /
  `core.Fail` / `core.Ok`. Never `fmt.Errorf`.
- **File I/O:** `coreio.Local` helpers; never `os.ReadFile/WriteFile`.
- **UK English; SPDX `EUPL-1.2` header on every file; conventional commits with
  `Co-Authored-By: Virgil <virgil@lethean.io>`.**
- **Tests:** `_Good/_Bad/_Ugly` + testify; live-VZ gated on `CONTAINER_VZ_LIVE=1`.
- **Registry/logs:** one shared inventory `~/.core/containers.json` +
  `~/.core/logs/{id}.log` across all providers.

## 6. Risks & open questions (resolve during per-SP specs)

- **R1 — guest image weight (SP3):** agent toolchains in a LinuxKit image may be large /
  slow to build. SP3 spec decides image caching strategy. (Workspace-delivery mechanism
  is now settled — virtio-fs host share; see SP3.)
- **R2 — secret injection ordering (SP3):** keys must reach the guest before the agent
  starts; vsock control handshake must precede agent launch.
- **R3 — protocol versioning (SP4):** host and `vzagent` ship together (RFC.vz.md §5),
  but the interactive-mode bump must not break the batch path used by SP2.
- **R4 — supply-chain gate timing (SP0a/SP1):** on darwin, `tmc/apple` cannot be isolated
  from detection (same package as `vz.go`), so **SP0(a) gates SP1's darwin merge** — not
  just SP2. The `//go:build vz` tag isolates only core-agent's own VZ code, not the
  transitive dependency; non-darwin builds stay clean.
- **R5 — fallback observability:** a silent VZ→docker downgrade must be visible in
  workspace status/logs so "why didn't it use VZ" is answerable.

---

## 7. Out of scope

- Linux/Windows VZ equivalents (VZ is Apple-only; those hosts use docker/podman).
- A hub HTTP `/container/:id/shell` websocket route (possible follow-up after SP4).
- Replacing the OCI-CLI path — it stays as the cross-platform fallback.
