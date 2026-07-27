// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"sync"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/process"
)

// VZ in-process dispatch fork.
//
// When the resolved runtime is `vz`, dispatch runs the agent in-process through
// go-container's Virtualization.framework provider instead of spawning an OCI
// `run --rm` process. This file is the fork: it builds the guest image + run
// options, boots the VM, drives the agent over the vsock control channel, and
// surfaces completion through the SAME agentCompletionMonitor the OCI path uses
// (via the vzCompletionProcess adapter satisfying completionProcess).
//
// SP3 wiring (this file):
//   - Host-visible workspace via virtio-fs: vzRunOptions shares workspaceDir
//     into the guest as the `workspace` tag (container.WithSharedDir), so the
//     agent's commits + BLOCKED.md land on the host. Unlike RunOptions.Volumes
//     (block-device images the guest must format), an FSShare is a live
//     directory the guest mounts at /workspace (guest-image responsibility, U2).
//   - Structured exec: the adapter drives container.VZProvider.ExecResult, which
//     preserves the real {stdout, stderr, exit} over the vsock control channel,
//     so onAgentComplete receives the true exit code (the lossy Exec folded a
//     non-zero exit into an error).
//   - Working directory + secret injection: vzAgentEnvCommand wraps the agent
//     command to run in /workspace/repo behind an existence guard (so the agent
//     operates on the checkout, matching the OCI `-w`), with an inline
//     `env K=V … <agent> <args>` carrying API keys + git identity from the host,
//     riding the vsock exec frame (not host-ps-visible; the guest
//     is hardware-isolated). A structured vzproto env verb would be cleaner —
//     future go-container work.

const (
	// vzImageEnv names the env var pointing at the §4 guest-image directory. When
	// set it is the OVERRIDE — vzResolveImage returns it directly and skips the
	// resolver. The directory must contain kernel + initrd.img (and optional
	// cmdline / disk.img). Unset, vzResolveImage shells the go-build resolver.
	vzImageEnv = "CORE_AGENT_VZ_IMAGE"
	// vzAgentBinEnv overrides the path to the cross-compiled VZ guest agent the
	// resolver bakes into the initrd. Default: <CoreRoot>/vz/vzagent.
	vzAgentBinEnv = "CORE_AGENT_VZAGENT_BIN"
	// vzCoreBinEnv overrides the name (or path) of the go-build `core` binary on
	// PATH that exposes `core build image-resolve`. Default: "core".
	vzCoreBinEnv = "CORE_BIN"
	// vzCoreBinDefault is the resolver binary name looked up on PATH when
	// CORE_BIN is unset.
	vzCoreBinDefault = "core"
	// vzDefaultMemoryMB is the guest memory allocation when dispatch config
	// carries none. go-container clamps to the framework's valid range.
	vzDefaultMemoryMB = 2048
	// vzDefaultCPUs is the guest vCPU count when dispatch config carries none.
	vzDefaultCPUs = 2
	// vzWorkspaceTag is the virtio-fs share tag for the host-visible workspace.
	// The guest image (U2) mounts it at /workspace, so the agent's commits +
	// BLOCKED.md land on the host directory.
	vzWorkspaceTag = "workspace"
	// vzGuestRepoDir is the in-guest working directory for the agent — the git
	// checkout under the /workspace mount (U2). The agent command runs here
	// (matching the OCI path's `-w /workspace/repo`); the `local` agent's
	// relative `-o ../.meta/agent-codex.log` resolves against it.
	vzGuestRepoDir = "/workspace/repo"
	// vzExitFailed is the exit code recorded when ExecResult fails at the verb
	// level (framework unavailable, container not running, transport error, or
	// an agent that refused the exec) — distinct from a command that ran and
	// exited non-zero, whose true code ExecResult preserves.
	vzExitFailed = 1
	// vzRuntimeName marks a WorkspaceStatus dispatched through the VZ fork. The
	// concurrency limiter counts these as running regardless of host PID (the VM
	// lives in-process, so there is no host child for ProcessAlive to find).
	vzRuntimeName = "vz"
)

// vzDispatcher is the minimal subset of *container.VZProvider the fork drives.
// Defined as an interface so unit tests inject a fake without booting a VM.
type vzDispatcher interface {
	// Available reports whether this host can boot VZ VMs (pre-Run gate).
	Available() bool
	// Run boots a guest image and returns the running *container.Container.
	Run(image *container.Image, opts ...container.RunOption) core.Result
	// ExecResult runs a command in the guest over vsock and returns its full
	// outcome — Value is a container.ExecResult{Stdout, Stderr, Exit}. A command
	// that ran and exited non-zero is OK at the verb level (the exit code is
	// preserved); only verb-level failures Fail.
	ExecResult(id, command string, args ...string) core.Result
	// Stop gracefully stops a running guest.
	Stop(id string) core.Result
}

// newVZProvider builds the dispatcher used by the fork. Overridden in tests to
// inject a fake; production returns the concrete in-process provider.
var newVZProvider = func() vzDispatcher { return container.NewVZProvider() }

// vzResolveExec runs the go-build image resolver and returns its core.Result —
// Value is the captured stdout string on success. It is a package var (a seam)
// so unit tests inject a scripted Result instead of shelling out. Production
// runs the command through the agent's process service (the same path spawnAgent
// uses), so a missing `core` binary, a non-zero exit, or a killed process all
// arrive here as result.OK == false.
var vzResolveExec = func(c *core.Core, ctx context.Context, bin string, args ...string) core.Result {
	return c.Process().Run(ctx, bin, args...)
}

// vzResolveImage builds the *container.Image the fork boots from.
//
// Resolution order:
//   - Override (CORE_AGENT_VZ_IMAGE set): return that directory directly, no
//     resolver — the stopgap / operator escape hatch.
//   - Default (env unset): shell the go-build resolver
//     `<CORE_BIN|core> build image-resolve --vzagent <bin> --output <dir>`,
//     which builds/caches a VZ guest kernel+initrd artefact and prints the
//     artefact directory alone on its last stdout line. The LAST non-empty
//     stdout line is taken as the image directory.
//
// Runtime deps for the default path (a clean error is returned, NOT a panic, if
// any are missing — spawnAgentVZ then falls back to the OCI path, U3):
//   - the go-build `core` binary on PATH (override its name via CORE_BIN);
//   - the cross-compiled `vzagent` guest agent at CORE_AGENT_VZAGENT_BIN, else
//     <CoreRoot>/vz/vzagent.
//
// The output/cache dir is a stable per-base directory under the runtime data
// root (<CoreRoot>/vz/guest/core-dev), so repeated dispatches reuse one cached
// artefact set rather than rebuilding per run.
//
// It is a package var (a seam) so unit tests swap the whole function; the
// resolver exec itself is the finer-grained vzResolveExec seam.
var vzResolveImage = func(c *core.Core) (*container.Image, error) {
	if dir := core.Trim(core.Env(vzImageEnv)); dir != "" {
		// Override path: trust the operator-supplied directory verbatim.
		return vzImageFor(dir), nil
	}

	vzagentBin := core.Trim(core.Env(vzAgentBinEnv))
	if vzagentBin == "" {
		vzagentBin = core.JoinPath(CoreRoot(), "vz", "vzagent")
	}
	if !fs.Exists(vzagentBin).OK {
		return nil, core.E("agentic.vzResolveImage", core.Concat("vzagent binary not found at ", vzagentBin, " (set ", vzAgentBinEnv, " or build the cross-compiled guest agent)"), nil)
	}

	coreBin := core.Trim(core.Env(vzCoreBinEnv))
	if coreBin == "" {
		coreBin = vzCoreBinDefault
	}
	outputDir := core.JoinPath(CoreRoot(), "vz", "guest", "core-dev")
	if ensureResult := fs.EnsureDir(outputDir); !ensureResult.OK {
		return nil, core.E("agentic.vzResolveImage", core.Concat("failed to create resolver output dir ", outputDir), forgeResultError(ensureResult))
	}

	result := vzResolveExec(c, context.Background(), coreBin, "build", "image-resolve", "--vzagent", vzagentBin, "--output", outputDir)
	if !result.OK {
		// Covers a `core` binary not on PATH, a non-zero exit, and a killed
		// process — Process().Run folds all three into result.OK == false.
		return nil, core.E("agentic.vzResolveImage", core.Concat(coreBin, " build image-resolve failed"), forgeResultError(result))
	}
	stdout, _ := result.Value.(string)
	dir := vzLastNonEmptyLine(stdout)
	if dir == "" {
		return nil, core.E("agentic.vzResolveImage", core.Concat(coreBin, " build image-resolve printed no artefact directory"), nil)
	}
	return vzImageFor(dir), nil
}

// vzImageFor builds the container.Image descriptor for a resolved guest-image
// directory — shared by the override and resolver paths so both produce the
// identical shape (a raw-format VZ image rooted at dir).
func vzImageFor(dir string) *container.Image {
	return &container.Image{
		Name:     "core-agent-vz",
		Path:     dir,
		Format:   container.FormatRaw,
		Provider: string(container.RuntimeVZ),
	}
}

// vzLastNonEmptyLine returns the last non-blank line of the resolver's stdout —
// the contract is that the artefact directory is printed alone on the final
// line, but build progress may precede it and a trailing newline may follow, so
// it scans backwards and skips blank lines rather than taking the raw last
// element. Returns "" when every line is blank.
func vzLastNonEmptyLine(output string) string {
	lines := core.Split(output, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if line := core.Trim(lines[i]); line != "" {
			return line
		}
	}
	return ""
}

// vzContainerID is the stable container name the fork assigns to a workspace's
// VM, so a later `core-agent shell` (SP4) can address it deterministically.
//
//	vzContainerID("/srv/core/workspace/core/go-io/task-5") // "vz-core-go-io-task-5"
func vzContainerID(workspaceDir string) string {
	return core.Concat("vz-", core.Replace(WorkspaceName(workspaceDir), "/", "-"))
}

// vzRunOptions maps dispatch config to go-container RunOptions: name, memory,
// cpus, and the host-visible workspace share. The workspace is shared via
// virtio-fs (container.WithSharedDir) rather than a block volume — an FSShare is
// a live host directory the guest mounts at /workspace (U2), so the agent's
// commits + BLOCKED.md land on the host. The meta dir is reachable as
// /workspace/.meta (it lives under workspaceDir), so no separate share is
// needed. API keys + git identity are NOT RunOptions.Env — they ride the exec
// frame via vzAgentEnvCommand (vsock, not host-ps-visible). dispatchMemory/
// dispatchCPUs default because DispatchConfig carries no such fields yet.
func (s *PrepSubsystem) vzRunOptions(workspaceDir string) []container.RunOption {
	return []container.RunOption{
		container.WithName(vzContainerID(workspaceDir)),
		container.WithMemory(vzDefaultMemoryMB),
		container.WithCPUs(vzDefaultCPUs),
		container.WithSharedDir(workspaceDir, vzWorkspaceTag),
	}
}

// vzCompletionProcess adapts an in-process VZ dispatch to the completionProcess
// contract (Done/Info/Output) so the existing agentCompletionMonitor +
// onAgentComplete machinery drives VZ exits unchanged. The VM is already booted
// by spawnAgentVZ (so Run-time entitlement/boot failures trigger the OCI
// fallback synchronously); a background goroutine runs only the Exec→Stop tail
// and records the outcome. Done closes when that tail finishes.
type vzCompletionProcess struct {
	id          string
	containerID string
	command     string
	args        []string
	startedAt   time.Time

	done chan struct{}

	mu     sync.Mutex
	info   process.Info
	output string
}

// run drives the post-boot VZ tail on a dispatched goroutine: exec the agent
// command over vsock with structured capture, record the true {stdout, stderr,
// exit}, then stop the (already running) VM. It always closes Done so the
// monitor never blocks and always attempts a stop so a booted VM never leaks.
// provider is passed in so spawnAgentVZ owns the seam wiring.
//
// The agent command is wrapped (vzAgentEnvCommand) so API keys + git identity
// ride the vsock exec frame as inline `env K=V …` — the guest is isolated and
// inherits no host env.
func (v *vzCompletionProcess) run(provider vzDispatcher) {
	defer close(v.done)
	// Always attempt a graceful stop once the agent command has run, even on a
	// failed exec — a booted VM must not leak.
	defer func() { _ = provider.Stop(v.containerID) }()

	envCommand, envArgs := vzAgentEnvCommand(v.command, v.args)
	execResult := provider.ExecResult(v.containerID, envCommand, envArgs...)
	if !execResult.OK {
		// Verb-level failure (framework unavailable, container not running,
		// transport error, or an agent that refused the exec) — distinct from a
		// command that ran and exited non-zero, which ExecResult reports as OK.
		v.finish(vzExitFailed, process.StatusFailed, vzResultMessage(execResult))
		return
	}
	result, ok := execResult.Value.(container.ExecResult)
	if !ok {
		v.finish(vzExitFailed, process.StatusFailed, "vz exec returned unexpected result type")
		return
	}
	// Preserve the true exit code + stderr. A non-zero exit is a failed agent
	// run; the monitor maps ExitCode → onAgentComplete unchanged.
	if result.Exit != 0 {
		v.finish(result.Exit, process.StatusFailed, vzExecOutput(result))
		return
	}
	v.finish(0, process.StatusExited, vzExecOutput(result))
}

// vzExecOutput combines a structured exec result into the single output string
// the completionProcess/monitor contract carries. stdout is the agent's
// captured output; stderr is appended (labelled) only when present so a failed
// run surfaces why without masking the stdout of a successful one.
func vzExecOutput(result container.ExecResult) string {
	if result.Stderr == "" {
		return result.Stdout
	}
	if result.Stdout == "" {
		return core.Concat("stderr: ", result.Stderr)
	}
	return core.Concat(result.Stdout, "\nstderr: ", result.Stderr)
}

// finish records the terminal outcome of the lifecycle under the lock.
func (v *vzCompletionProcess) finish(exitCode int, status process.Status, output string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.output = output
	v.info = process.Info{
		ID:        v.id,
		Command:   v.command,
		Args:      v.args,
		StartedAt: v.startedAt,
		Running:   false,
		Status:    status,
		ExitCode:  exitCode,
		Duration:  time.Since(v.startedAt),
		PID:       vzSentinelPID,
	}
}

// Done reports lifecycle completion to the monitor.
func (v *vzCompletionProcess) Done() <-chan struct{} { return v.done }

// Info returns the recorded process info (terminal values once Done fires).
func (v *vzCompletionProcess) Info() process.Info {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.info
}

// Output returns the captured agent stdout.
func (v *vzCompletionProcess) Output() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.output
}

// vzSentinelPID is the host PID reported for a VZ dispatch. The VM lives inside
// this process, so there is no child PID — -1 is the honest "no host process"
// sentinel. A pid<=0 makes ProcessAlive report dead, so the concurrency limiter
// cannot count a VZ dispatch by PID; instead WorkspaceStatus.Runtime=="vz"
// (recorded by recordVZRuntime, carried by preserveStatusNote) makes
// countRunningByAgent/countRunningByModel count it as running. Completion is
// unaffected — it runs off the vzCompletionProcess Done channel, not ProcessAlive.
const vzSentinelPID = -1

// vzAgentEnvVar names one host env var injected into the guest agent command and
// the optional default applied when the host value is empty. keys with no
// default are dropped when unset (no point exporting an empty API key).
type vzAgentEnvVar struct {
	name        string
	hostFrom    string // host env var to read; defaults to name when empty
	defaultWhen string // value used when the host value is empty ("" = drop)
}

// vzAgentEnvVars is the secret + git-identity set injected into the guest agent
// command. API keys are dropped when unset; git identity always has a Virgil
// default so commits inside the guest are attributable. The host is the source
// of truth — the guest inherits no environment, so each value must be passed.
var vzAgentEnvVars = []vzAgentEnvVar{
	{name: "OPENAI_API_KEY"},
	{name: "ANTHROPIC_API_KEY"},
	{name: "GEMINI_API_KEY"},
	{name: "GOOGLE_API_KEY"},
	{name: "GIT_AUTHOR_NAME", defaultWhen: "Virgil"},
	{name: "GIT_COMMITTER_NAME", defaultWhen: "Virgil"},
	{name: "GIT_AUTHOR_EMAIL", defaultWhen: "virgil@lethean.io"},
	{name: "GIT_COMMITTER_EMAIL", defaultWhen: "virgil@lethean.io"},
}

// vzAgentEnvCommand wraps the agent command so it runs in the right guest
// directory with API keys + git identity in scope. It returns ("sh", ["-c",
// "if [ ! -d /workspace/repo ]; …; cd /workspace/repo && env K=V … <agent>
// <args>"]):
//   - a guard that fails fast if the workspace share did not mount (the agent
//     would otherwise run against an empty / wrong tree),
//   - cd into /workspace/repo, so the agent operates on the git checkout and the
//     `local` agent's relative `-o ../.meta/agent-codex.log` resolves (this
//     mirrors the OCI path's `-w /workspace/repo` + existence guard, making the
//     command self-sufficient rather than depending on the guest exec verb's
//     default cwd),
//   - inline env: the values (read from the host via core.Env, shell-quoted)
//     ride the vsock exec frame (§5), so they are never visible to host `ps` and
//     the guest is hardware-isolated. Empty API keys are omitted; git identity
//     defaults to Virgil.
//
// A structured vzproto verb (carrying cwd + env alongside the command) would be
// cleaner than inline shell — future go-container work; until then the inline
// form mirrors the OCI path's shell-script wrapping.
//
//	cmd, args := vzAgentEnvCommand("codex", []string{"exec", "--full-auto"})
//	// "sh", ["-c", "if [ ! -d /workspace/repo ]; …; cd /workspace/repo && env OPENAI_API_KEY='…' … 'codex' 'exec' '--full-auto'"]
func vzAgentEnvCommand(command string, args []string) (string, []string) {
	script := core.NewBuilder()
	// Fail fast if the workspace share did not mount — an agent run against a
	// missing checkout produces confusing failures far from the cause.
	script.WriteString(core.Concat("if [ ! -d ", vzGuestRepoDir, " ]; then echo 'missing ", vzGuestRepoDir, "' >&2; exit 1; fi; "))
	script.WriteString(core.Concat("cd ", vzGuestRepoDir, " && env"))
	for _, spec := range vzAgentEnvVars {
		hostKey := spec.hostFrom
		if hostKey == "" {
			hostKey = spec.name
		}
		value := core.Trim(core.Env(hostKey))
		if value == "" {
			value = spec.defaultWhen
		}
		if value == "" {
			continue // unset API key — nothing to export
		}
		script.WriteString(core.Concat(" ", spec.name, "=", shellQuote(value)))
	}
	script.WriteString(core.Concat(" ", shellQuote(command)))
	for _, arg := range args {
		script.WriteString(core.Concat(" ", shellQuote(arg)))
	}
	return "sh", []string{"-c", script.String()}
}

// vzResultMessage extracts a human-readable message from a failed core.Result.
func vzResultMessage(result core.Result) string {
	if err, ok := result.Value.(error); ok && err != nil {
		return err.Error()
	}
	return "vz dispatch failed"
}

// spawnAgentVZ is the in-process fork of spawnAgent for the `vz` runtime. It
// mirrors spawnAgent's (pid, processID, outputFile, error) contract plus a
// fellBack flag. It boots the VM SYNCHRONOUSLY so every Run-time failure — the
// framework being unavailable, the image being unresolvable, OR the entitlement
// error the framework only raises at Run (IsVZAvailable can be true while the
// binary is unentitled, RFC.vz.md §2.2) — is a fallback trigger: it records a
// VZ→OCI downgrade Note on the workspace status (SP2.4 / R5 observability) and
// returns fellBack=true so the caller takes the OCI path. Only once the VM is
// running does it hand the container to the completion adapter for the Exec→Stop
// tail and wire the existing monitor.
//
//	pid, pid0, out, fellBack, err := s.spawnAgentVZ(agent, cmd, args, ws, meta, outFile)
func (s *PrepSubsystem) spawnAgentVZ(agent, command string, args []string, workspaceDir, _ /* metaDir */, outputFile string) (int, string, string, bool, error) {
	provider := newVZProvider()
	if provider == nil || !provider.Available() {
		s.recordVZDowngrade(workspaceDir, agent, "Virtualization.framework unavailable")
		return 0, "", outputFile, true, nil
	}

	image, err := vzResolveImage(s.Core())
	if err != nil {
		s.recordVZDowngrade(workspaceDir, agent, "VZ guest image unavailable: "+err.Error())
		return 0, "", outputFile, true, nil
	}

	// Boot synchronously: the entitlement error is only knowable from Run, so a
	// failed boot must fall back here, not surface later as a failed agent run.
	runResult := provider.Run(image, s.vzRunOptions(workspaceDir)...)
	if !runResult.OK {
		s.recordVZDowngrade(workspaceDir, agent, "VZ boot failed: "+vzResultMessage(runResult))
		return 0, "", outputFile, true, nil
	}
	ctr, ok := runResult.Value.(*container.Container)
	if !ok || ctr == nil {
		s.recordVZDowngrade(workspaceDir, agent, "VZ boot returned no container")
		return 0, "", outputFile, true, nil
	}

	monitorProcess := &vzCompletionProcess{
		id:          vzContainerID(workspaceDir),
		containerID: ctr.ID,
		command:     command,
		args:        args,
		startedAt:   time.Now(),
		done:        make(chan struct{}),
	}

	// Tag the workspace as VZ-dispatched BEFORE the monitor goroutine starts: the
	// concurrency limiter counts a vz workspace as running regardless of host PID
	// (vzSentinelPID is not a real OS process), and the caller's post-spawn write
	// carries this forward via preserveStatusNote. Writing it after `go run` would
	// risk reverting a fast `completed` write back to `running`.
	s.recordVZRuntime(workspaceDir, agent)

	go monitorProcess.run(provider)

	s.broadcastStart(agent, workspaceDir)
	s.startIssueTracking(workspaceDir)

	monitorAction := core.Concat("agentic.monitor.", core.Replace(WorkspaceName(workspaceDir), "/", "."))
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        agent,
		workspaceDir: workspaceDir,
		outputFile:   outputFile,
		process:      monitorProcess,
	}
	s.Core().Action(monitorAction, monitor.run)
	if result := s.Core().PerformAsync(monitorAction, core.NewOptions()); !result.OK {
		return 0, "", outputFile, false, core.E("dispatch.spawnAgentVZ", "failed to start monitor", forgeResultError(result))
	}

	return vzSentinelPID, monitorProcess.id, outputFile, false, nil
}

// preserveStatusNote carries the VZ annotations recorded inside spawnAgent —
// the VZ→OCI downgrade Note (recordVZDowngrade) and the Runtime tag
// (recordVZRuntime) — across a caller's post-spawn status write. Several callers
// build a fresh WorkspaceStatus (or reuse a struct read before the spawn) and
// write it to record the pid/processID, which would otherwise clobber the
// on-disk Note/Runtime. Each field is carried forward only when the new status
// leaves it empty — touching exactly the empty fields, so it cannot disturb
// existing write semantics.
//
// Scaffold caveat: on a reused workspace (queue resume, Runs++), an annotation
// from a PRIOR run can persist into a later run of a different shape. Threading
// these through spawnAgent's return would avoid this but cascades a 6-caller
// signature change — not worth it for the env-gated fork.
//
//	preserveStatusNote(workspaceDir, freshStatus) // before writeStatusResult
func preserveStatusNote(workspaceDir string, status *WorkspaceStatus) {
	if status == nil {
		return
	}
	if status.Note != "" && status.Runtime != "" {
		return
	}
	prev, ok := workspaceStatusValue(ReadStatusResult(workspaceDir))
	if !ok {
		return
	}
	if status.Note == "" && prev.Note != "" {
		status.Note = prev.Note
	}
	if status.Runtime == "" && prev.Runtime != "" {
		status.Runtime = prev.Runtime
	}
}

// recordVZRuntime tags the workspace status as VZ-dispatched on the success path,
// so the concurrency limiter counts the dispatch as running despite the sentinel
// PID (the VM has no host child for ProcessAlive to find). Like recordVZDowngrade
// it must be durable before the agent completes and before the caller's
// post-spawn write — on the primary dispatch path status.json may not exist yet
// (the caller writes it only after spawnAgent returns), so a missing status is
// created with a minimal running record. The caller's later write preserves the
// Runtime via preserveStatusNote. Best-effort: a failed write is logged, not
// fatal (a dropped tag only under-counts, never mis-dispatches).
func (s *PrepSubsystem) recordVZRuntime(workspaceDir, agent string) {
	workspaceStatus, ok := workspaceStatusValue(ReadStatusResult(workspaceDir))
	if !ok {
		workspaceStatus = &WorkspaceStatus{Status: "running", Agent: agent, StartedAt: time.Now()}
	}
	workspaceStatus.Runtime = vzRuntimeName
	if writeResult := writeStatusResult(workspaceDir, workspaceStatus); !writeResult.OK {
		core.Warn("agentic.spawnAgentVZ: failed to record vz runtime tag", "reason", writeResult.Error())
	}
}

// recordVZDowngrade annotates the workspace status with a VZ→OCI downgrade note
// so the fallback is observable (SP2.4 / R5). The note must be durable on the
// primary dispatch path, where prepWorkspace has NOT yet written status.json when
// the fallback fires (the caller writes it only after spawnAgent returns). So a
// missing status is created with a minimal running record carrying the note,
// rather than dropped. The caller's later write then preserves it via
// preserveStatusNote. Best-effort: a failed write is logged, not fatal.
func (s *PrepSubsystem) recordVZDowngrade(workspaceDir, agent, reason string) {
	note := core.Concat("runtime downgraded vz→oci: ", reason)
	core.Warn("agentic.spawnAgentVZ: "+note, "workspace", WorkspaceName(workspaceDir))
	workspaceStatus, ok := workspaceStatusValue(ReadStatusResult(workspaceDir))
	if !ok {
		// No status.json yet (fresh dispatch path) — create a minimal coherent
		// record so the downgrade is observable before the OCI agent completes.
		workspaceStatus = &WorkspaceStatus{Status: "running", Agent: agent, StartedAt: time.Now()}
	}
	workspaceStatus.Note = note
	if writeResult := writeStatusResult(workspaceDir, workspaceStatus); !writeResult.OK {
		core.Warn("agentic.spawnAgentVZ: failed to record downgrade note", "reason", writeResult.Error())
	}
}
