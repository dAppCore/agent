// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/container"
	"dappco.re/go/process"
)

// fakeVZDispatcher is an injectable stand-in for *container.VZProvider so the
// fork's unit tests never boot a VM. Each verb's result is scripted; calls are
// recorded so tests can assert the Run→Exec→Stop ordering.
type fakeVZDispatcher struct {
	available  bool
	runResult  core.Result
	execResult core.Result
	stopResult core.Result

	runCalls  int
	execCalls int
	stopCalls int

	lastRunOpts     container.RunOptions
	lastExecCommand string
	lastExecArgs    []string
}

func (f *fakeVZDispatcher) Available() bool { return f.available }

func (f *fakeVZDispatcher) Run(image *container.Image, opts ...container.RunOption) core.Result {
	f.runCalls++
	f.lastRunOpts = container.ApplyRunOptions(opts...)
	return f.runResult
}

func (f *fakeVZDispatcher) ExecResult(id, command string, args ...string) core.Result {
	f.execCalls++
	f.lastExecCommand = command
	f.lastExecArgs = args
	return f.execResult
}

func (f *fakeVZDispatcher) Stop(id string) core.Result {
	f.stopCalls++
	return f.stopResult
}

// withFakeVZProvider swaps newVZProvider for the test and restores it after.
func withFakeVZProvider(t *testing.T, fake vzDispatcher) {
	t.Helper()
	previous := newVZProvider
	newVZProvider = func() vzDispatcher { return fake }
	t.Cleanup(func() { newVZProvider = previous })
}

// withFakeVZImage swaps vzResolveImage so spawnAgentVZ proceeds past the image
// gate without an on-disk §4 artefact directory.
func withFakeVZImage(t *testing.T, image *container.Image, err error) {
	t.Helper()
	previous := vzResolveImage
	vzResolveImage = func() (*container.Image, error) { return image, err }
	t.Cleanup(func() { vzResolveImage = previous })
}

// --- runtimeUsesProvider / resolveOCIRuntime (fork routing) ---

func TestDispatchVZ_RuntimeUsesProvider_Good_Case(t *testing.T) {
	core.AssertTrue(t, runtimeUsesProvider(RuntimeVZ))
	core.AssertFalse(t, runtimeUsesProvider(RuntimeDocker))
	core.AssertFalse(t, runtimeUsesProvider(RuntimeApple))
}

func TestDispatchVZ_ResolveOCIRuntime_Good_Case(t *testing.T) {
	// The fallback landing target is never vz — it has no argv form.
	resolved := resolveOCIRuntime()
	core.AssertNotEqual(t, RuntimeVZ, resolved)
	core.AssertContains(t, []string{RuntimeApple, RuntimeDocker, RuntimePodman}, resolved)
}

// --- vzDispatchEnabled (SP2.1) ---

func TestDispatchVZ_DispatchEnabled_Bad_NonDarwinOrUnset(t *testing.T) {
	// With the live opt-in unset, the gate is always closed regardless of host.
	t.Setenv("CONTAINER_VZ_LIVE", "")
	core.AssertFalse(t, vzDispatchEnabled())
}

func TestDispatchVZ_DispatchEnabled_Ugly_OptInButFrameworkGates(t *testing.T) {
	// Opt-in alone is not enough — IsVZAvailable() must also be true. On a CI
	// host (no Apple silicon / framework) the gate stays closed even with the
	// env set, which is exactly the safe default.
	t.Setenv("CONTAINER_VZ_LIVE", "1")
	if !container.IsVZAvailable() {
		core.AssertFalse(t, vzDispatchEnabled())
	} else {
		core.AssertTrue(t, vzDispatchEnabled())
	}
}

// --- vzContainerID ---

func TestDispatchVZ_ContainerID_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	id := vzContainerID(core.JoinPath(root, "core", "go-io", "task-5"))
	core.AssertContains(t, id, "vz-")
	core.AssertNotContains(t, id, "/")
}

// --- vzRunOptions (SP3.1: maps memory/cpus/name + virtio-fs workspace share) ---

func TestDispatchVZ_RunOptions_Good_WorkspaceShare(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	s := &PrepSubsystem{}
	workspaceDir := core.JoinPath(root, "core", "go-io", "task-5")
	opts := s.vzRunOptions(workspaceDir)
	applied := container.ApplyRunOptions(opts...)

	core.AssertEqual(t, vzDefaultMemoryMB, applied.Memory)
	core.AssertEqual(t, vzDefaultCPUs, applied.CPUs)
	core.AssertContains(t, applied.Name, "vz-")
	// SP3: the workspace is shared host-visible via virtio-fs (a live directory),
	// NOT a block volume (VZ volumes are raw image FILES the guest must format).
	core.RequireTrue(t, len(applied.FSShares) == 1)
	core.AssertEqual(t, workspaceDir, applied.FSShares[0].HostDir)
	core.AssertEqual(t, vzWorkspaceTag, applied.FSShares[0].Tag)
	core.AssertFalse(t, applied.FSShares[0].ReadOnly) // workspace is RW — commits land on host
	core.AssertEqual(t, 0, len(applied.Volumes))      // no block volumes
	// API keys + git identity ride the exec frame (vzAgentEnvCommand), not Env.
	core.AssertEqual(t, 0, len(applied.Env))
}

// --- vzCompletionProcess (the completionProcess adapter) ---

func TestDispatchVZ_CompletionProcess_Good_ExecStop(t *testing.T) {
	// The VM is already booted (spawnAgentVZ Runs synchronously); the adapter
	// drives only the structured ExecResult→Stop tail.
	fake := &fakeVZDispatcher{
		available:  true,
		execResult: core.Ok(container.ExecResult{Stdout: "agent stdout", Exit: 0}),
		stopResult: core.Ok(nil),
	}
	proc := &vzCompletionProcess{
		id:          "vz-test",
		containerID: "vzfake01",
		command:     "true",
		startedAt:   time.Now(),
		done:        make(chan struct{}),
	}

	proc.run(fake)
	<-proc.Done() // closed by run

	core.AssertEqual(t, 0, fake.runCalls) // adapter never Runs — boot is upstream
	core.AssertEqual(t, 1, fake.execCalls)
	core.AssertEqual(t, 1, fake.stopCalls) // VM stopped even on success
	core.AssertEqual(t, "agent stdout", proc.Output())
	core.AssertEqual(t, 0, proc.Info().ExitCode)
	core.AssertEqual(t, process.StatusExited, proc.Info().Status)
	// Sentinel PID — the VM lives in-process, no host child.
	core.AssertEqual(t, vzSentinelPID, proc.Info().PID)
	// The agent command is wrapped with inline env over the exec frame.
	core.AssertEqual(t, "sh", fake.lastExecCommand)
	core.RequireTrue(t, len(fake.lastExecArgs) == 2)
	core.AssertEqual(t, "-c", fake.lastExecArgs[0])
	core.AssertContains(t, fake.lastExecArgs[1], "env ")
	core.AssertContains(t, fake.lastExecArgs[1], "'true'") // shell-quoted agent command
}

func TestDispatchVZ_CompletionProcess_Ugly_ExecVerbFails(t *testing.T) {
	// A verb-level ExecResult failure (framework unavailable, container not
	// running, transport error, agent refused) → failed run, synthetic exit, VM
	// still stopped.
	fake := &fakeVZDispatcher{
		available:  true,
		execResult: core.Fail(core.E("VZProvider.ExecResult", "agent refused exec: vsock closed", nil)),
		stopResult: core.Ok(nil),
	}
	proc := &vzCompletionProcess{id: "vz-test", containerID: "vzfake01", command: "false", startedAt: time.Now(), done: make(chan struct{})}

	proc.run(fake)
	<-proc.Done()

	core.AssertEqual(t, 1, fake.execCalls)
	core.AssertEqual(t, 1, fake.stopCalls) // VM stopped despite exec failure
	core.AssertEqual(t, vzExitFailed, proc.Info().ExitCode)
	core.AssertEqual(t, process.StatusFailed, proc.Info().Status)
}

func TestDispatchVZ_CompletionProcess_Ugly_NonZeroExitPreserved(t *testing.T) {
	// Structured exec: a command that RAN and exited non-zero is OK at the verb
	// level. The adapter must surface the TRUE exit code (not the synthetic
	// vzExitFailed) and fold stderr into the output for the monitor.
	fake := &fakeVZDispatcher{
		available:  true,
		execResult: core.Ok(container.ExecResult{Stdout: "partial", Stderr: "boom", Exit: 2}),
		stopResult: core.Ok(nil),
	}
	proc := &vzCompletionProcess{id: "vz-test", containerID: "vzfake01", command: "false", startedAt: time.Now(), done: make(chan struct{})}

	proc.run(fake)
	<-proc.Done()

	core.AssertEqual(t, 1, fake.execCalls)
	core.AssertEqual(t, 1, fake.stopCalls)
	core.AssertEqual(t, 2, proc.Info().ExitCode) // real exit code, not vzExitFailed
	core.AssertEqual(t, process.StatusFailed, proc.Info().Status)
	core.AssertContains(t, proc.Output(), "partial")
	core.AssertContains(t, proc.Output(), "boom") // stderr surfaced
}

// --- completion adapter drives onAgentComplete (end-to-end via the monitor) ---

func TestDispatchVZ_CompletionDrivesOnAgentComplete_Good_Case(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)

	wsDir := core.JoinPath(root, "ws-vz")
	repoDir := core.JoinPath(wsDir, "repo")
	metaDir := core.JoinPath(wsDir, ".meta")
	fs.EnsureDir(repoDir)
	fs.EnsureDir(metaDir)

	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", PID: vzSentinelPID, StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	// A real vzCompletionProcess driven by a fake provider — proving the adapter
	// satisfies completionProcess AND that the existing monitor consumes it.
	fake := &fakeVZDispatcher{available: true, execResult: core.Ok(container.ExecResult{Stdout: "vz output", Exit: 0}), stopResult: core.Ok(nil)}
	proc := &vzCompletionProcess{id: "vz-ws", containerID: "vzfake01", command: "true", startedAt: time.Now(), done: make(chan struct{})}
	proc.run(fake)

	s := newPrepWithProcess()
	monitor := &agentCompletionMonitor{
		service:      s,
		agent:        "codex",
		workspaceDir: wsDir,
		outputFile:   core.JoinPath(metaDir, "agent-codex.log"),
		process:      proc,
	}
	r := monitor.run(context.Background(), core.NewOptions())
	core.AssertTrue(t, r.OK)

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, "completed", updated.Status)
	core.AssertEqual(t, 0, updated.PID) // onAgentComplete clears PID
	out := fs.Read(core.JoinPath(metaDir, "agent-codex.log"))
	core.RequireTrue(t, out.OK)
	core.AssertEqual(t, "vz output", out.Value.(string))
}

// --- spawnAgentVZ auto-fallback (SP2.4) ---

func TestDispatchVZ_SpawnFallback_Good_ProviderUnavailable(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-fallback")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	// Provider reports unavailable → fork must fall back BEFORE any boot, and
	// before any s.Core() use (so a bare PrepSubsystem is safe here).
	withFakeVZProvider(t, &fakeVZDispatcher{available: false})
	s := &PrepSubsystem{}

	pid, processID, outputFile, fellBack, err := s.spawnAgentVZ("codex", "true", nil, wsDir, WorkspaceMetaDir(wsDir), "out.log")
	core.AssertNoError(t, err)
	core.AssertTrue(t, fellBack)
	core.AssertEqual(t, 0, pid)
	core.AssertEqual(t, "", processID)
	core.AssertEqual(t, "out.log", outputFile)

	// R5: the downgrade is observable on the workspace status.
	updated := mustReadStatus(t, wsDir)
	core.AssertContains(t, updated.Note, "vz→oci")
}

func TestDispatchVZ_SpawnFallback_Bad_ImageUnavailable(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-noimage")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	// Provider available, but no guest image resolvable → fall back with a note.
	withFakeVZProvider(t, &fakeVZDispatcher{available: true})
	withFakeVZImage(t, nil, core.E("dispatch.vz", "CORE_AGENT_VZ_IMAGE is not set", nil))
	s := &PrepSubsystem{}

	_, _, _, fellBack, err := s.spawnAgentVZ("codex", "true", nil, wsDir, WorkspaceMetaDir(wsDir), "out.log")
	core.AssertNoError(t, err)
	core.AssertTrue(t, fellBack)

	updated := mustReadStatus(t, wsDir)
	core.AssertContains(t, updated.Note, "guest image unavailable")
}

// SP2.4: IsVZAvailable()==true while the binary is unentitled — the framework
// only raises the entitlement error from Run. The synchronous boot must catch
// it, fall back to OCI, and never reach Exec. This is the precise case the
// gate-on-available design (SP2.1) relies on SP2.4 to handle.
func TestDispatchVZ_SpawnFallback_Ugly_RunEntitlementError(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-entitlement")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	st := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(st))

	fake := &fakeVZDispatcher{
		available: true,
		runResult: core.Fail(core.E("VZProvider.Run", "validate configuration: com.apple.security.virtualization entitlement required", nil)),
	}
	withFakeVZProvider(t, fake)
	withFakeVZImage(t, &container.Image{Path: t.TempDir()}, nil)
	s := &PrepSubsystem{}

	_, _, _, fellBack, err := s.spawnAgentVZ("codex", "true", nil, wsDir, WorkspaceMetaDir(wsDir), "out.log")
	core.AssertNoError(t, err)
	core.AssertTrue(t, fellBack)
	core.AssertEqual(t, 1, fake.runCalls)  // boot attempted synchronously
	core.AssertEqual(t, 0, fake.execCalls) // never execs a VM that did not boot

	updated := mustReadStatus(t, wsDir)
	core.AssertContains(t, updated.Note, "vz→oci")
	core.AssertContains(t, updated.Note, "boot failed")
}

// On the primary dispatch path, prepWorkspace has NOT written status.json when
// the fallback fires (the caller writes it only after spawnAgent returns). The
// downgrade must still be observable — recordVZDowngrade creates a minimal status
// rather than dropping the note. This test deliberately does NOT pre-seed
// status.json, unlike the _SpawnFallback_* tests above.
func TestDispatchVZ_SpawnFallback_Ugly_NoPriorStatusFile(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-nostatus")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	// No status.json written — fresh dispatch path.
	core.AssertFalse(t, fs.Exists(core.JoinPath(wsDir, "status.json")))

	withFakeVZProvider(t, &fakeVZDispatcher{available: false})
	s := &PrepSubsystem{}

	_, _, _, fellBack, err := s.spawnAgentVZ("codex", "true", nil, wsDir, WorkspaceMetaDir(wsDir), "out.log")
	core.AssertNoError(t, err)
	core.AssertTrue(t, fellBack)

	// The note was created from nothing — observable even without prepWorkspace.
	updated := mustReadStatus(t, wsDir)
	core.AssertContains(t, updated.Note, "vz→oci")
	core.AssertEqual(t, "codex", updated.Agent)
	core.AssertEqual(t, "running", updated.Status)
}

// --- preserveStatusNote (SP2.4 Note survives the caller's post-spawn write) ---

// The downgrade Note recorded inside spawnAgent must survive the caller's
// post-spawn fresh-struct write (dispatch.go / queue.go / resume.go), or the R5
// observability promise is broken before anyone reads it. Reproduces that exact
// sequence: on-disk Note → fresh struct → preserveStatusNote → write → read.
func TestDispatchVZ_PreserveStatusNote_Good_SurvivesFreshWrite(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-note")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))

	// recordVZDowngrade wrote this during the fallback inside spawnAgent.
	downgraded := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", Note: "runtime downgraded vz→oci: VZ boot failed", StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(downgraded))

	// The caller then builds a fresh struct to record the OCI pid (Note unset).
	fresh := &WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: 4242, ProcessID: "proc-1", StartedAt: time.Now(), Runs: 1}
	preserveStatusNote(wsDir, fresh)
	writeStatusResult(wsDir, fresh)

	updated := mustReadStatus(t, wsDir)
	core.AssertContains(t, updated.Note, "vz→oci")
	core.AssertEqual(t, 4242, updated.PID) // the fresh write still took effect
}

// A status that explicitly carries its own Note is never overwritten by a stale
// on-disk one (the helper only fills an empty Note).
func TestDispatchVZ_PreserveStatusNote_Ugly_DoesNotOverrideExplicit(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-note2")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Note: "old note"}))

	fresh := &WorkspaceStatus{Status: "running", Note: "explicit note"}
	preserveStatusNote(wsDir, fresh)
	core.AssertEqual(t, "explicit note", fresh.Note)
}

// The Runtime tag recorded inside spawnAgentVZ (recordVZRuntime) must survive
// the caller's post-spawn fresh-struct write, or the concurrency limiter never
// sees a VZ dispatch as running (SP3.4). Mirrors the Note carry, on the Runtime
// field: on-disk Runtime → fresh struct → preserveStatusNote → write → read.
func TestDispatchVZ_PreserveStatusNote_Good_RuntimeSurvivesFreshWrite(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-runtime")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))

	// recordVZRuntime tagged this during the VZ success path inside spawnAgent.
	tagged := &WorkspaceStatus{Status: "running", Repo: "go-io", Agent: "codex", Runtime: vzRuntimeName, PID: vzSentinelPID, StartedAt: time.Now()}
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(tagged))

	// The caller then builds a fresh struct to record pid/runs (Runtime unset).
	fresh := &WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", PID: vzSentinelPID, StartedAt: time.Now(), Runs: 1}
	preserveStatusNote(wsDir, fresh)
	writeStatusResult(wsDir, fresh)

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, vzRuntimeName, updated.Runtime)
}

// preserveStatusNote carries Note and Runtime independently — a fresh write that
// sets one but not the other still inherits the missing field from disk.
func TestDispatchVZ_PreserveStatusNote_Ugly_CarriesNoteAndRuntimeIndependently(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-both")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Note: "disk note", Runtime: vzRuntimeName}))

	// Fresh write carries an explicit Runtime but no Note → keep its Runtime,
	// inherit the Note.
	fresh := &WorkspaceStatus{Status: "running", Runtime: "oci-explicit"}
	preserveStatusNote(wsDir, fresh)
	core.AssertEqual(t, "disk note", fresh.Note)       // inherited
	core.AssertEqual(t, "oci-explicit", fresh.Runtime) // not overwritten
}

// --- recordVZRuntime (create-or-update on the success path) ---

// On the primary dispatch path status.json does not exist when spawnAgentVZ
// runs; recordVZRuntime must create a minimal running record carrying the tag
// rather than dropping it (same create-or-update as recordVZDowngrade).
func TestDispatchVZ_RecordVZRuntime_Good_CreatesWhenNoStatus(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-rt-create")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	core.AssertFalse(t, fs.Exists(core.JoinPath(wsDir, "status.json")))

	s := &PrepSubsystem{}
	s.recordVZRuntime(wsDir, "codex")

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, vzRuntimeName, updated.Runtime)
	core.AssertEqual(t, "running", updated.Status)
	core.AssertEqual(t, "codex", updated.Agent)
}

// When status.json already exists, recordVZRuntime updates the Runtime field in
// place without clobbering the rest of the record.
func TestDispatchVZ_RecordVZRuntime_Good_UpdatesExisting(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	wsDir := core.JoinPath(root, "ws-rt-update")
	fs.EnsureDir(core.JoinPath(wsDir, ".meta"))
	fs.Write(core.JoinPath(wsDir, "status.json"), core.JSONMarshalString(&WorkspaceStatus{Status: "running", Agent: "codex", Repo: "go-io", Branch: "feat/x"}))

	s := &PrepSubsystem{}
	s.recordVZRuntime(wsDir, "codex")

	updated := mustReadStatus(t, wsDir)
	core.AssertEqual(t, vzRuntimeName, updated.Runtime)
	core.AssertEqual(t, "feat/x", updated.Branch) // existing fields preserved
}

// --- vzAgentEnvCommand (secret + git-identity injection over the exec frame) ---

func TestDispatchVZ_AgentEnvCommand_Good_GitDefaultsAndKey(t *testing.T) {
	// One API key set, the rest unset → only the set key is exported; git
	// identity always carries the Virgil default.
	t.Setenv("OPENAI_API_KEY", "sk-test-123")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("GIT_AUTHOR_NAME", "")
	t.Setenv("GIT_COMMITTER_NAME", "")
	t.Setenv("GIT_AUTHOR_EMAIL", "")
	t.Setenv("GIT_COMMITTER_EMAIL", "")

	command, args := vzAgentEnvCommand("codex", []string{"exec", "--full-auto"})
	core.AssertEqual(t, "sh", command)
	core.RequireTrue(t, len(args) == 2)
	core.AssertEqual(t, "-c", args[0])
	script := args[1]

	// Set key exported (shell-quoted); unset keys omitted entirely.
	core.AssertContains(t, script, "OPENAI_API_KEY='sk-test-123'")
	core.AssertNotContains(t, script, "ANTHROPIC_API_KEY=")
	core.AssertNotContains(t, script, "GEMINI_API_KEY=")
	core.AssertNotContains(t, script, "GOOGLE_API_KEY=")
	// Git identity defaults applied.
	core.AssertContains(t, script, "GIT_AUTHOR_NAME='Virgil'")
	core.AssertContains(t, script, "GIT_COMMITTER_NAME='Virgil'")
	core.AssertContains(t, script, "GIT_AUTHOR_EMAIL='virgil@lethean.io'")
	core.AssertContains(t, script, "GIT_COMMITTER_EMAIL='virgil@lethean.io'")
	// Agent command + args appended, shell-quoted, after the env assignments.
	core.AssertContains(t, script, "'codex' 'exec' '--full-auto'")
	// The command runs in the guest repo dir behind an existence guard (matches
	// the OCI `-w /workspace/repo` + guard), so the agent operates on the
	// checkout and relative output paths resolve.
	core.AssertContains(t, script, "if [ ! -d /workspace/repo ]")
	core.AssertContains(t, script, "cd /workspace/repo && env ")
	core.AssertTrue(t, core.HasPrefix(script, "if [ ! -d /workspace/repo ]"))
}

func TestDispatchVZ_AgentEnvCommand_Good_HostGitIdentityOverridesDefault(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "Snider")
	t.Setenv("GIT_AUTHOR_EMAIL", "snider@host.uk.com")

	_, args := vzAgentEnvCommand("claude", nil)
	core.RequireTrue(t, len(args) == 2)
	script := args[1]
	core.AssertContains(t, script, "GIT_AUTHOR_NAME='Snider'")
	core.AssertContains(t, script, "GIT_AUTHOR_EMAIL='snider@host.uk.com'")
	core.AssertNotContains(t, script, "GIT_AUTHOR_NAME='Virgil'")
}

// A value containing a single quote must be shell-quoted safely so the script
// cannot break out of the quoting (defence against injection via env/args).
func TestDispatchVZ_AgentEnvCommand_Ugly_ShellQuotesUnsafeValue(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "ab'cd")

	_, args := vzAgentEnvCommand("codex", []string{"weird'arg"})
	core.RequireTrue(t, len(args) == 2)
	script := args[1]
	// shellQuote turns ' into '\'' — the raw unescaped sequence must not appear.
	core.AssertContains(t, script, `OPENAI_API_KEY='ab'\''cd'`)
	core.AssertContains(t, script, `'weird'\''arg'`)
}

// --- vzResolveImage production behaviour ---

func TestDispatchVZ_ResolveImage_Bad_EnvUnset(t *testing.T) {
	t.Setenv(vzImageEnv, "")
	image, err := vzResolveImage()
	core.AssertError(t, err)
	core.AssertNil(t, image)
}

func TestDispatchVZ_ResolveImage_Good_EnvSet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(vzImageEnv, dir)
	image, err := vzResolveImage()
	core.AssertNoError(t, err)
	core.AssertNotNil(t, image)
	core.AssertEqual(t, dir, image.Path)
}
