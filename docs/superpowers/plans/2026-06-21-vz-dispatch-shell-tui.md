<!-- SPDX-License-Identifier: EUPL-1.2 -->

# VZ-first Containerised Dispatch + Container Shell TUI — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Spec:** `docs/superpowers/specs/2026-06-21-vz-dispatch-shell-tui-design.md`

**Goal:** Make core-agent dispatch agents on Apple Virtualization.framework (in-process, via go-container's `VZProvider`) in place of Docker when available, and add a `core-agent shell <id>` route that drops the user into an interactive shell inside a running container/VM.

**Architecture:** core-agent imports `dappco.re/go/container` and forks the dispatch execution path — OCI runtimes (docker/apple/podman) keep the existing `run --rm -v` argv path; `vz` calls the concrete `*VZProvider` in-process. Runtime detection routes through go-container's `Detect()` (priority apple→vz→docker→podman→linuxkit). VZ is "best available" — a signed/entitled build boots VMs; everything else auto-falls-back to apple→docker.

**Tech Stack:** Go 1.26.2; `dappco.re/go` (core), `dappco.re/go/container` (providers), `dappco.re/go/process`; `github.com/tmc/apple` + `ebitengine/purego` (transitive, darwin, no-cgo); LinuxKit (guest images, SP3).

## Global Constraints

- **Module resolution:** siblings are versioned modules — **no `go.work`, no `replace`**. Add deps with `go get dappco.re/go/container@<ver>` then `go mod tidy`. Build/test from the `go/` dir. CI: `GOWORK=off GOFLAGS=-mod=mod`. Proxy auth via `GONOSUMCHECK=dappco.re/*,forge.lthn.ai/*`.
- **Errors:** `core.E("pkg.Method", "message", err)` (always 3 args) / `core.Result{Value, OK}` / `core.Fail` / `core.Ok`. **Never** `fmt.Errorf`.
- **File I/O:** `coreio.Local` / `fs.*` helpers. **Never** `os.ReadFile`/`os.WriteFile`.
- **Tests:** `TestX_Behaviour_{Good,Bad,Ugly}` using the in-repo `core.Assert*` helpers (match `pkg/agentic/dispatch_runtime_test.go`). Live-VZ tests gate on `CONTAINER_VZ_LIVE=1` AND a signed/entitled binary.
- **Style:** UK English; `// SPDX-License-Identifier: EUPL-1.2` first line of every Go file; conventional commits `type(scope): desc` ending `Co-Authored-By: Virgil <virgil@lethean.io>`.
- **Shared state:** one registry `~/.core/containers.json` + logs `~/.core/logs/{id}.log` across all providers (go-container owns both).
- **VZ entitlement:** VZ verbs fail at `ValidateWithError()` without `com.apple.security.virtualization`; treat that error as a **fallback trigger**, never a panic/hard-fail.

---

## Phase Roadmap

| Phase | Deliverable | Implementable now? | Gate / depends |
|-------|-------------|--------------------|----------------|
| **SP0** | Operator gates: `tmc/apple` supply-chain review + signing/entitlement | Yes (non-code, operator) | Blocks SP1 **darwin merge** + SP2/SP4 |
| **SP1** | go-container dep + detection seam + `vz` recognised (no boot path yet) | **Yes — fully specified below** | SP0(a) for darwin merge |
| **SP2** | VZ in-process dispatch fork (boot/exec/stop, auto-fallback) | Yes — specified below | SP1 |
| **SP3** | LinuxKit agent-guest-image pipeline + **go-container virtio-fs workspace share** | **Needs its own spec first** | SP1; RFC.vz.md §4 update |
| **SP4** | vsock PTY protocol + vzagent PTY + `core-agent shell <id>` | **Needs its own spec first** | SP3 (image), SP1/SP2; RFC.vz.md §5 update |
| **SP5** | Specced-but-incomplete cleanup (Metal GPU wire-through, GOAL-STATUS remainders) | Yes — checklist below | independent |

> **Why SP3/SP4 are not bite-sized here:** writing "complete code in every step" for an undesigned guest-image pipeline or a new wire protocol would be fabrication. The brainstorming spec already marks both "own spec". This plan implements **SP0–SP2 + SP5** to executable detail and defines SP3/SP4 as phases that each run their own brainstorming→spec→writing-plans cycle. Do SP3 before SP4 (SP4's `vzagent` ships inside SP3's image).

---

## SP0 — Operator gates (non-code; run in parallel; blocks merge not dev)

**Owner:** operator (Snider/Hades-scope). No Go tasks.

- [ ] **SP0.1 — Supply-chain review** of `github.com/tmc/apple` (`virtualization` + `x/vzkit` only; **never** `private/*`) and `ebitengine/purego`. Pin exact versions (`tmc/apple v0.6.12` is the version VZProvider currently builds against). Vendoring acceptable if the review prefers it. Record sign-off. **This must clear before SP1 merges on darwin** — importing go-container's `container` package transitively compiles `vz.go`→`tmc/apple` on darwin (see SP1.1 note).
- [ ] **SP0.2 — Signing + entitlement.** Add `com.apple.security.virtualization` to the core-agent release codesign step; provision the signing identity in the release pipeline. Acceptance: a signed release binary boots a VZ VM on an Apple-silicon host; an unsigned `go build` does not (and falls back per SP2).

**Done when:** both sign-offs recorded and the entitlement round-trips on a live host.

---

## SP1 — go-container dependency + detection seam

**Outcome:** detection routes through go-container; `vz` is a recognised runtime + config value; **the OCI dispatch path (docker/apple/podman) is byte-for-byte unchanged**; `vz` is NOT yet auto-selected (no boot path until SP2).

**Files:**
- Modify: `go/go.mod`, `go/go.sum`
- Create: `go/pkg/agentic/runtime_container.go` (the detection seam)
- Modify: `go/pkg/agentic/dispatch.go` (add `RuntimeVZ`; re-point `runtimeAvailable`; guard `vz` out of auto until SP2)
- Modify: `go/pkg/runner/queue.go` (doc the `vz` value on `DispatchConfig.Runtime`)
- Test: `go/pkg/agentic/runtime_container_test.go`, extend `go/pkg/agentic/dispatch_runtime_test.go`

**Interfaces:**
- Consumes (from go-container): `container.Detect() container.ContainerRuntime`, `container.DetectAll() []container.ContainerRuntime`, `container.HasRuntime(container.RuntimeType) bool`, constants `container.RuntimeApple/RuntimeVZ/RuntimeDocker/RuntimePodman/RuntimeLinuxKit/RuntimeNone`, field `ContainerRuntime.Type container.RuntimeType`.
- Produces (for SP2): `RuntimeVZ = "vz"` const in `agentic`; `containerRuntimeAvailable(name string) bool`; `vzDispatchEnabled() bool` (false in SP1, flipped in SP2); `runtimeUsesProvider(name string) bool` (true for `vz`).

### Task SP1.1 — Add the go-container dependency + detection smoke test

- [ ] **Step 1: Add the module.** From the `go/` dir:

```bash
cd go && GONOSUMCHECK=dappco.re/*,forge.lthn.ai/* go get dappco.re/go/container@latest && go mod tidy
```

- [ ] **Step 2: Write the smoke test** `go/pkg/agentic/runtime_container_test.go`:

```go
// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
	"dappco.re/go/container"
)

// Detect always returns a runtime record (RuntimeNone when nothing is found)
// — never panics, never an empty Type.
func TestRuntimeContainer_Detect_Good(t *testing.T) {
	rt := container.Detect()
	core.AssertNotEmpty(t, string(rt.Type))
}
```

- [ ] **Step 3: Build + run.**

```bash
cd go && go build ./... && go test ./pkg/agentic/ -run TestRuntimeContainer_Detect_Good -count=1
```
Expected: build succeeds; test PASS.

> **Gate note:** on darwin this compiles `tmc/apple` transitively (go-container's `Detect()` shares a package with darwin-only `vz.go`). Do not merge to a release branch until **SP0.1** clears. For local dev before sign-off, this builds and runs fine.

- [ ] **Step 4: Commit.**

```bash
git add go/go.mod go/go.sum go/pkg/agentic/runtime_container_test.go
git commit -m "feat(agentic): add dappco.re/go/container dependency + detection smoke test" -m "Co-Authored-By: Virgil <virgil@lethean.io>"
```

### Task SP1.2 — Detection seam: route availability through go-container

- [ ] **Step 1: Write the failing test** (append to `runtime_container_test.go`):

```go
// Docker/podman availability via the seam agrees with go-container's HasRuntime.
func TestRuntimeContainer_Available_Good(t *testing.T) {
	core.AssertEqual(t, container.HasRuntime(container.RuntimeDocker), containerRuntimeAvailable("docker"))
	core.AssertEqual(t, container.HasRuntime(container.RuntimePodman), containerRuntimeAvailable("podman"))
}

// Unknown runtimes are never available through the seam.
func TestRuntimeContainer_Available_Bad(t *testing.T) {
	core.AssertFalse(t, containerRuntimeAvailable(""))
	core.AssertFalse(t, containerRuntimeAvailable("kubernetes"))
}
```

- [ ] **Step 2: Run — expect FAIL** (`containerRuntimeAvailable` undefined):

```bash
cd go && go test ./pkg/agentic/ -run TestRuntimeContainer_Available -count=1
```

- [ ] **Step 3: Create the seam + add the `RuntimeVZ` const.** The seam below references `RuntimeVZ`, and SP1.3's resolver in turn needs this seam — a mutual compile-time dependency. So add `RuntimeVZ = "vz"` to the runtime const block in `go/pkg/agentic/dispatch.go` (after `RuntimeApple`) in THIS task; it is a behaviourless identifier, and SP1.3 adds only the guard logic. Then create `go/pkg/agentic/runtime_container.go`:

```go
// SPDX-License-Identifier: EUPL-1.2

package agentic

import "dappco.re/go/container"

// containerRuntimeAvailable reports whether a runtime is usable on this host,
// delegating to go-container's detection (single source of truth, replaces the
// old $PATH probe). Unknown names are never available.
//
//	containerRuntimeAvailable("docker") // true if dockerd reachable
func containerRuntimeAvailable(name string) bool {
	switch name {
	case RuntimeApple, RuntimeVZ, RuntimeDocker, RuntimePodman:
		return container.HasRuntime(container.RuntimeType(name))
	default:
		return false
	}
}

// runtimeUsesProvider reports whether a runtime is driven through go-container's
// in-process provider (vz) rather than the OCI argv path (docker/apple/podman).
//
//	runtimeUsesProvider("vz") // true
func runtimeUsesProvider(name string) bool { return name == RuntimeVZ }

// vzDispatchEnabled gates whether `auto` may resolve to vz. SP1 keeps it OFF so
// the OCI path is unchanged; SP2 flips it on once the boot fork exists.
func vzDispatchEnabled() bool { return false }
```

- [ ] **Step 4: Run — expect PASS.**

```bash
cd go && go test ./pkg/agentic/ -run TestRuntimeContainer_Available -count=1
```

- [ ] **Step 5: Commit.**

```bash
git add go/pkg/agentic/runtime_container.go go/pkg/agentic/runtime_container_test.go
git commit -m "feat(agentic): detection seam delegating runtime availability to go-container" -m "Co-Authored-By: Virgil <virgil@lethean.io>"
```

### Task SP1.3 — Add the `vz` runtime constant + keep it out of `auto`

- [ ] **Step 1: Write the failing test** (append to `dispatch_runtime_test.go`):

```go
// vz is a recognised constant but, in SP1, never auto-selected (no boot path).
func TestDispatchRuntime_VZ_NotAutoSelected_Good(t *testing.T) {
	core.AssertEqual(t, "vz", RuntimeVZ)
	// auto must never surface vz until SP2 enables the fork.
	core.AssertNotEqual(t, RuntimeVZ, resolveContainerRuntime(RuntimeAuto))
}

// An explicit vz preference, with the fork disabled, falls back to an OCI runtime.
func TestDispatchRuntime_VZ_ExplicitFallsBack_Ugly(t *testing.T) {
	resolved := resolveContainerRuntime(RuntimeVZ)
	core.AssertNotEqual(t, RuntimeVZ, resolved)
	core.AssertContains(t, []string{RuntimeApple, RuntimeDocker, RuntimePodman}, resolved)
}
```

- [ ] **Step 2: Run — expect FAIL** (`RuntimeVZ` undefined):

```bash
cd go && go test ./pkg/agentic/ -run TestDispatchRuntime_VZ -count=1
```

- [ ] **Step 3: Add the guard** in `go/pkg/agentic/dispatch.go`. (The `RuntimeVZ = "vz"` const was already added in SP1.2 — the seam references it, so it could not wait until here. Do not re-add it.)

Change `resolveContainerRuntime` so the auto-order includes vz only when enabled, and an explicit `vz` with the fork off falls through to OCI. Replace the body (note: the availability calls go through `runtimeAvailable`, the single apple-policy + seam entry point — see the SP1.4 note):

```go
func resolveContainerRuntime(preferred string) string {
	if preferred == RuntimeVZ && !vzDispatchEnabled() {
		preferred = RuntimeAuto // fork not ready — fall through to OCI
	}
	switch preferred {
	case RuntimeApple, RuntimeVZ, RuntimeDocker, RuntimePodman:
		if runtimeAvailable(preferred) {
			return preferred
		}
	}
	order := []string{RuntimeApple}
	if vzDispatchEnabled() {
		order = append(order, RuntimeVZ)
	}
	order = append(order, RuntimeDocker, RuntimePodman)
	for _, candidate := range order {
		if runtimeAvailable(candidate) {
			return candidate
		}
	}
	return RuntimeDocker
}
```

- [ ] **Step 4: Run — expect PASS** (and re-run the whole runtime suite to prove no OCI regression):

```bash
cd go && go test ./pkg/agentic/ -run 'TestDispatchRuntime' -count=1
```
Expected: all PASS (existing `_ResolveContainerRuntime_*`, `_ContainerCommandFor_*` still green).

- [ ] **Step 5: Commit.**

```bash
git add go/pkg/agentic/dispatch.go go/pkg/agentic/dispatch_runtime_test.go
git commit -m "feat(agentic): recognise vz runtime, guarded out of auto until SP2" -m "Co-Authored-By: Virgil <virgil@lethean.io>"
```

### Task SP1.4 — Point `runtimeAvailable` at the seam (single detection source)

> **As-built note:** `runtimeAvailable` now both delegates to the seam (`containerRuntimeAvailable`) AND is the function `resolveContainerRuntime` calls (per the SP1.3 resolver above), so it is the single live detection entry point — not dead code. SP1.4 and SP1.3 were reconciled in a follow-up cleanup commit; do not also leave `resolveContainerRuntime` calling the seam directly.

- [ ] **Step 1: Run the existing availability tests to capture current green:**

```bash
cd go && go test ./pkg/agentic/ -run 'TestDispatchRuntime_RuntimeAvailable' -count=1
```
Expected: PASS (baseline before refactor).

- [ ] **Step 2: Re-point `runtimeAvailable`** in `dispatch.go` to delegate, preserving the apple-on-non-darwin=false rule:

```go
func runtimeAvailable(name string) bool {
	if name == RuntimeApple && !goosIsDarwin {
		return false
	}
	return containerRuntimeAvailable(name)
}
```

Remove the now-dead `containerRuntimeBinary` PATH-probe usage only if nothing else calls it — `containerCommandFor` still needs `containerRuntimeBinary` for the OCI argv, so **keep `containerRuntimeBinary`**.

- [ ] **Step 3: Run — expect PASS** (existing `_RuntimeAvailable_*` + full runtime suite):

```bash
cd go && go test ./pkg/agentic/ -run 'TestDispatchRuntime' -count=1 && go vet ./...
```

- [ ] **Step 4: Commit.**

```bash
git add go/pkg/agentic/dispatch.go
git commit -m "refactor(agentic): runtimeAvailable delegates to the go-container seam" -m "Co-Authored-By: Virgil <virgil@lethean.io>"
```

### Task SP1.5 — Document the `vz` config value

- [ ] **Step 1:** In `go/pkg/runner/queue.go`, update the `DispatchConfig.Runtime` doc comment to list `vz`:

```go
	// Runtime selects the container runtime — auto | apple | vz | docker | podman.
	// auto detects in preference order: Apple Container -> VZ (when enabled) ->
	// Docker -> Podman. vz uses the in-process Virtualization.framework provider.
	Runtime string `yaml:"runtime"`
```

- [ ] **Step 2: Build + full package test:**

```bash
cd go && go build ./... && go test ./pkg/agentic/ ./pkg/runner/ -count=1
```
Expected: PASS.

- [ ] **Step 3: Commit.**

```bash
git add go/pkg/runner/queue.go
git commit -m "docs(runner): document vz as a dispatch.runtime value" -m "Co-Authored-By: Virgil <virgil@lethean.io>"
```

**SP1 done when:** `go test ./pkg/agentic/ ./pkg/runner/` is green, detection flows through go-container, `vz` is a recognised config value, and `auto` still resolves to apple/docker (no behaviour change). SP0.1 cleared before merging the darwin build.

---

## SP2 — VZ in-process dispatch fork

**Outcome:** when the resolved runtime is `vz`, dispatch boots a VM via the concrete `*VZProvider` and runs the agent through its vsock `Exec`, tracked in the shared registry; entitlement failures auto-fall-back to apple→docker. Flip `vzDispatchEnabled()` to true.

**Files:**
- Modify: `go/pkg/agentic/runtime_container.go` (`vzDispatchEnabled` → entitlement/opt-in aware)
- Create: `go/pkg/agentic/dispatch_vz.go` (the in-process fork: build `*Image`+`RunOption`s, Run, Exec, Stop, fallback)
- Modify: `go/pkg/agentic/dispatch.go` (at the spawn call-site ~`:712`, branch `runtimeUsesProvider(rt)` → `dispatch_vz.go`, else existing argv)
- Test: `go/pkg/agentic/dispatch_vz_test.go`

**Interfaces:**
- Consumes (from go-container): `container.NewVZProvider() *container.VZProvider`; methods `(*VZProvider).Available() bool`, `.Run(image *container.Image, opts ...container.RunOption) core.Result` (Value `*container.Container`), `.Exec(id, cmd string, args ...string) core.Result` (Value string), `.Stop(id) core.Result`, `.Kill(id) core.Result`, `.Logs(id string, tail int) core.Result`, `.Wait(ctx, id) core.Result`; options `container.WithMemory(mb int)`, `WithCPUs(n)`, `WithVolumes(map[string]string)`, `WithEnv(...string)`, `WithName(string)`.
- Consumes (from SP1): `runtimeUsesProvider`, `vzDispatchEnabled`.
- Produces (for SP4): `vzContainerID(workspaceDir string) string` (stable id used for `core-agent shell`).

**Key task outline** (each a TDD cycle following the SP1 pattern):

- [ ] **SP2.1 — `vzDispatchEnabled` becomes real:** true only when `container.IsVZAvailable()` AND (entitled OR `CONTAINER_VZ_LIVE=1`). Tests: false on non-darwin; false when env unset and unentitled. *Note:* entitlement can't be probed pre-`Run` (RFC.vz.md §2.2) — treat "available" as the gate and rely on SP2.4 runtime fallback.
- [ ] **SP2.2 — image + options builder** in `dispatch_vz.go`: map `dispatchImage()`→`*container.Image` (resolve to the guest-artefact dir SP3 produces; until SP3, accept a `CORE_AGENT_VZ_IMAGE` dir for live tests), and dispatch config → `[]container.RunOption` (memory/cpus/volumes=workspace+meta/env=API keys/name). Pure construction — unit-testable without boot.
- [ ] **SP2.3 — the fork** at `dispatch.go` spawn site: `if runtimeUsesProvider(rt) { return s.dispatchVZ(...) }` else existing argv. `dispatchVZ` calls `VZProvider.Run`, records the `*Container` in workspace status + shared registry, streams logs. Test with a fake provider seam (inject an interface so the unit test doesn't boot).
- [ ] **SP2.4 — auto-fallback:** when `Run` returns an error naming the missing entitlement (or any VZ-unavailable error), retry down apple→docker and record the downgrade in `WorkspaceStatus` (R5 observability). Test: fake provider returns entitlement error → asserts OCI path taken + status notes downgrade.
- [ ] **SP2.5 — live boot (gated):** `//go:build vz` + `CONTAINER_VZ_LIVE=1` test that boots `CORE_AGENT_VZ_IMAGE`, execs `true`, stops. Skipped everywhere by default.

**SP2 done when:** on a signed/entitled host with a minimal VZ image, `dispatch.runtime: vz` boots, execs, and registers; unentitled/CI hosts fall back cleanly with the downgrade visible in status; non-live tests green via the injected provider seam.

---

## SP3 — LinuxKit agent-guest-image pipeline  *(write its own spec first)*

**Status:** Needs a brainstorming→spec→writing-plans cycle of its own. The spec (`docs/superpowers/specs/<date>-sp3-vz-guest-image.md`) must settle: image contents, caching, and the **go-container virtio-fs change**.

**Scope (for that spec to expand):**
- LinuxKit YAML → kernel+initrd+rootfs with toolchains (node/go/python) + agent CLIs (codex/claude/gemini) + `vzagent` service + `CONFIG_VIRTIO_VSOCKETS=y` + `CAP_SYS_BOOT`.
- **go-container change:** add a `VZVirtioFileSystemDeviceConfiguration` directory-share to `VZProvider` (host workspace dir, tagged via `NewVirtioFileSystemDeviceConfigurationWithTag`), guest mounts the tag rw. `tmc/apple v0.6.12` + `x/vzkit/virtiofs` already expose this. **Extends RFC.vz.md §4.**
- Spec baking (`~/spec/` read-only) per core-agent RFC §15.5.2.
- Secret/git-identity injection over vsock before agent launch (R2 ordering).
- `build.linuxkit.resolve("core-dev"|"core-ml"|"core-minimal")` action (RFC §15.5.3) → cached bootable artefact set; replaces SP2.2's `CORE_AGENT_VZ_IMAGE` stopgap.

**Acceptance:** `build.linuxkit.resolve("core-dev")` yields an artefact set whose guest runs an agent against a **host-visible** workspace (commits land on the host repo) with injected keys.

---

## SP4 — Interactive shell: vsock PTY + `core-agent shell <id>`  *(write its own spec first)*

**Status:** Needs its own spec (`docs/superpowers/specs/<date>-sp4-vz-pty-shell.md`). `vzproto` is batch-only today; the interactive protocol is a new design. **Extends RFC.vz.md §5.**

**Scope (for that spec to expand):**
- **go-container — `vzproto` interactive mode:** framed session — `open(cols,rows)`, bidirectional stdin/stdout data frames, `resize(cols,rows)`, `exit(code)`; keep the batch protocol intact; bump a protocol version. Unit-test over `net.Pipe` (no VM).
- **go-container — `vzagent` PTY:** allocate a PTY (`creack/pty` or raw syscall), attach the shell, pump both directions, honour resize/exit. Reship the static binary; **SP3's image must bake this `vzagent`** (hence SP3 before SP4).
- **core-agent — `core-agent shell <id>`** CLI: raw-mode local terminal; VZ → dial control vsock, `open`, multiplex `os.Stdin`↔stdout, `SIGWINCH`→`resize`, restore on exit; docker/podman → `<rt> exec -it <id> $SHELL`; apple → reuse `AppleProvider.ExecInteractive(id, cmd...)`. Reuse `pkg/opencode/tui.go` quoting helpers (`shellQuote`/`appleScriptQuote`/`cmdArgvQuote`) for argv safety. Register the subcommand in `cmd/core-agent/main.go`.

**Acceptance:** `core-agent shell <id>` gives a working interactive shell into a running OCI container AND a running VZ VM, with working resize and clean exit.

---

## SP5 — Specced-but-incomplete cleanup (checklist)

- [ ] **Metal GPU wire-through:** thread `dispatchGPU()` → `container.WithGPU(true)` on the VZ path; map `ContainerRuntime.HasGPU()` into `dispatchGPU` capability checks. No-op until Apple's framework exposes Metal passthrough (RFC.vz.md §15, RFC §15.5.3) — but the option + capability plumb end-to-end with a test asserting the no-op today.
- [ ] **go-container GOAL-STATUS remainders** (track upstream, not in this repo): macOS 26+ CLI-flag verification; AX polish audit; RFC §3.3 AMI/GCP formats; v0.9.0 audit findings; RFC cross-reference resolution. File as go-container tickets; reference them here.

**SP5 done when:** GPU option plumbs with a passing no-op test; remainder items are filed as go-container tickets with links recorded.

---

## Self-Review

**Spec coverage** (spec §3 SP0–SP5 → tasks): SP0 → SP0.1/0.2 ✓; SP1 → SP1.1–1.5 ✓ (full TDD); SP2 → SP2.1–2.5 ✓ (task-level with interfaces); SP3 → phase + own-spec pointer ✓ (intentionally not bite-sized — undesigned); SP4 → phase + own-spec pointer ✓; SP5 → checklist ✓. Spec §2.3 auto-fallback → SP2.4 ✓. Spec §4.1 go-container-side work → SP3 (virtio-fs)/SP4 (PTY) ✓. Spec §6 risks: R1 settled in SP3; R2 SP3 secret-injection; R3 SP4 protocol-version; R4 SP0.1↔SP1.1 gate note ✓; R5 SP2.4 downgrade-observability ✓.

**Placeholder scan:** SP3/SP4 are deliberately phase-level (own spec) — flagged explicitly, not hidden placeholders. SP1/SP2 carry real code/commands. No "TBD"/"add error handling"/"similar to" left in SP1.

**Type consistency:** `RuntimeVZ`/`containerRuntimeAvailable`/`runtimeUsesProvider`/`vzDispatchEnabled` defined in SP1.2/SP1.3, consumed unchanged in SP2. go-container signatures (`Detect`/`HasRuntime`/`NewVZProvider`/`Run`/`Exec`/`WithMemory`…) match what was read from `provider.go`/`runtime.go`/`vz.go`. `containerRuntimeBinary` kept (OCI argv still needs it) — noted in SP1.4.
