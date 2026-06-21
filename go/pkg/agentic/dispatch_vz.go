// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"sync"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/process"
)

// SP2 — VZ in-process dispatch fork (scaffold).
//
// When the resolved runtime is `vz`, dispatch runs the agent in-process through
// go-container's Virtualization.framework provider instead of spawning an OCI
// `run --rm` process. This file is the fork: it builds the guest image + run
// options, boots the VM, drives the agent over the vsock control channel, and
// surfaces completion through the SAME agentCompletionMonitor the OCI path uses
// (via the vzCompletionProcess adapter satisfying completionProcess).
//
// Scaffold scope (SP3 supersedes): the workspace is NOT yet host-visible inside
// the guest. go-container's RunOptions.Volumes are block-device attachments —
// vzVolumeSpecs requires each source to be a raw image FILE, so passing the
// workspace directory would make Run fail on every dispatch. Host-visible
// workspace sharing (virtio-fs) and secret/git-identity injection over vsock are
// SP3. SP2 therefore boots a minimal VM (memory/cpus/name only) to prove the
// fork plumbing end-to-end.
//
// Exec limitation (flagged for SP3): container.VZProvider.Exec returns only
// stdout on exit==0 and folds a non-zero exit into a core.Fail error — it does
// not surface a structured {stdout, stderr, exit}. The adapter therefore maps
// Ok→exit 0 and Fail→exit 1 (failed). Real agent dispatch needs a structured
// exec verb from go-container.

const (
	// vzImageEnv names the env var pointing at the §4 guest-image directory used
	// until SP3's build.linuxkit.resolve pipeline produces it. The directory
	// must contain kernel + initrd.img (and optional cmdline / disk.img).
	vzImageEnv = "CORE_AGENT_VZ_IMAGE"
	// vzDefaultMemoryMB is the guest memory allocation when dispatch config
	// carries none. go-container clamps to the framework's valid range.
	vzDefaultMemoryMB = 2048
	// vzDefaultCPUs is the guest vCPU count when dispatch config carries none.
	vzDefaultCPUs = 2
	// vzExitFailed is the synthetic exit code recorded when the guest agent
	// reports a non-zero exit (go-container folds the real code into an error;
	// SP3's structured exec will surface the true value).
	vzExitFailed = 1
)

// vzDispatcher is the minimal subset of *container.VZProvider the fork drives.
// Defined as an interface so unit tests inject a fake without booting a VM.
type vzDispatcher interface {
	// Available reports whether this host can boot VZ VMs (pre-Run gate).
	Available() bool
	// Run boots a guest image and returns the running *container.Container.
	Run(image *container.Image, opts ...container.RunOption) core.Result
	// Exec runs a command in the guest over vsock and returns its stdout.
	Exec(id, command string, args ...string) core.Result
	// Stop gracefully stops a running guest.
	Stop(id string) core.Result
}

// newVZProvider builds the dispatcher used by the fork. Overridden in tests to
// inject a fake; production returns the concrete in-process provider.
var newVZProvider = func() vzDispatcher { return container.NewVZProvider() }

// vzResolveImage builds the *container.Image the fork boots from. It is a seam
// (package var) so unit tests bypass the on-disk §4 artefact check. Production
// resolves the guest-image directory from CORE_AGENT_VZ_IMAGE (SP3 replaces this
// with the build.linuxkit.resolve artefact set).
var vzResolveImage = func() (*container.Image, error) {
	dir := core.Trim(core.Env(vzImageEnv))
	if dir == "" {
		return nil, core.E("dispatch.vz", vzImageEnv+" is not set (no VZ guest image)", nil)
	}
	return &container.Image{
		Name:     "core-agent-vz",
		Path:     dir,
		Format:   container.FormatRaw,
		Provider: string(container.RuntimeVZ),
	}, nil
}

// vzContainerID is the stable container name the fork assigns to a workspace's
// VM, so a later `core-agent shell` (SP4) can address it deterministically.
//
//	vzContainerID("/srv/core/workspace/core/go-io/task-5") // "vz-core-go-io-task-5"
func vzContainerID(workspaceDir string) string {
	return core.Concat("vz-", core.Replace(WorkspaceName(workspaceDir), "/", "-"))
}

// vzRunOptions maps dispatch config to go-container RunOptions. SCAFFOLD: only
// memory/cpus/name. Workspace+meta volumes and API-key env are deliberately
// omitted — see the file header (volumes are block-device-only; env is SP3
// vsock injection). dispatchMemory/dispatchCPUs default because DispatchConfig
// carries no such fields yet.
func (s *PrepSubsystem) vzRunOptions(workspaceDir string) []container.RunOption {
	return []container.RunOption{
		container.WithName(vzContainerID(workspaceDir)),
		container.WithMemory(vzDefaultMemoryMB),
		container.WithCPUs(vzDefaultCPUs),
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
// command over vsock, capture stdout/exit, then stop the (already running) VM.
// It always closes Done so the monitor never blocks and always attempts a stop
// so a booted VM never leaks. provider is passed in so spawnAgentVZ owns the
// seam wiring.
func (v *vzCompletionProcess) run(provider vzDispatcher) {
	defer close(v.done)
	// Always attempt a graceful stop once the agent command has run, even on a
	// failed exec — a booted VM must not leak.
	defer func() { _ = provider.Stop(v.containerID) }()

	execResult := provider.Exec(v.containerID, v.command, v.args...)
	if !execResult.OK {
		// go-container folds a non-zero guest exit into a Fail error; treat any
		// exec failure as a failed agent run (SP3 structured exec surfaces the
		// real code + stderr).
		v.finish(vzExitFailed, process.StatusFailed, vzResultMessage(execResult))
		return
	}
	stdout, _ := execResult.Value.(string)
	v.finish(0, process.StatusExited, stdout)
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
// sentinel. NOTE: unlike a real OS PID, this does NOT make the dispatch count as
// running in countRunningByAgent (ProcessAlive treats pid<=0 with no processID
// as dead); the concurrency limiter therefore under-counts in-flight VZ agents.
// Completion is unaffected — it runs off the vzCompletionProcess Done channel,
// not ProcessAlive. Accurate in-flight accounting is an SP3 concern.
const vzSentinelPID = -1

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
		s.recordVZDowngrade(workspaceDir, "Virtualization.framework unavailable")
		return 0, "", outputFile, true, nil
	}

	image, err := vzResolveImage()
	if err != nil {
		s.recordVZDowngrade(workspaceDir, "VZ guest image unavailable: "+err.Error())
		return 0, "", outputFile, true, nil
	}

	// Boot synchronously: the entitlement error is only knowable from Run, so a
	// failed boot must fall back here, not surface later as a failed agent run.
	runResult := provider.Run(image, s.vzRunOptions(workspaceDir)...)
	if !runResult.OK {
		s.recordVZDowngrade(workspaceDir, "VZ boot failed: "+vzResultMessage(runResult))
		return 0, "", outputFile, true, nil
	}
	ctr, ok := runResult.Value.(*container.Container)
	if !ok || ctr == nil {
		s.recordVZDowngrade(workspaceDir, "VZ boot returned no container")
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

// recordVZDowngrade annotates the workspace status with a VZ→OCI downgrade note
// so the fallback is observable (SP2.4 / R5). Best-effort: a missing or
// unreadable status is logged, not fatal — the OCI path still runs.
func (s *PrepSubsystem) recordVZDowngrade(workspaceDir, reason string) {
	note := core.Concat("runtime downgraded vz→oci: ", reason)
	core.Warn("agentic.spawnAgentVZ: "+note, "workspace", WorkspaceName(workspaceDir))
	result := ReadStatusResult(workspaceDir)
	workspaceStatus, ok := workspaceStatusValue(result)
	if !ok {
		return
	}
	workspaceStatus.Note = note
	if writeResult := writeStatusResult(workspaceDir, workspaceStatus); !writeResult.OK {
		core.Warn("agentic.spawnAgentVZ: failed to record downgrade note", "reason", writeResult.Error())
	}
}
