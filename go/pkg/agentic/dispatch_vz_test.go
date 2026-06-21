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
	available bool
	runResult core.Result
	execResult core.Result
	stopResult core.Result

	runCalls  int
	execCalls int
	stopCalls int

	lastRunOpts container.RunOptions
}

func (f *fakeVZDispatcher) Available() bool { return f.available }

func (f *fakeVZDispatcher) Run(image *container.Image, opts ...container.RunOption) core.Result {
	f.runCalls++
	f.lastRunOpts = container.ApplyRunOptions(opts...)
	return f.runResult
}

func (f *fakeVZDispatcher) Exec(id, command string, args ...string) core.Result {
	f.execCalls++
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

// --- vzRunOptions (SP2.2: scaffold maps memory/cpus/name, NOT volumes) ---

func TestDispatchVZ_RunOptions_Good_NoWorkspaceVolume(t *testing.T) {
	root := t.TempDir()
	setTestWorkspace(t, root)
	s := &PrepSubsystem{}
	opts := s.vzRunOptions(core.JoinPath(root, "core", "go-io", "task-5"))
	applied := container.ApplyRunOptions(opts...)

	core.AssertEqual(t, vzDefaultMemoryMB, applied.Memory)
	core.AssertEqual(t, vzDefaultCPUs, applied.CPUs)
	core.AssertContains(t, applied.Name, "vz-")
	// SP3 gap: the workspace is a directory, and VZ volumes are block-device
	// FILES (vzVolumeSpecs requires IsFile(source)). The scaffold must NOT map
	// the workspace as a volume — doing so would fail Run on every dispatch.
	core.AssertEqual(t, 0, len(applied.Volumes))
	core.AssertEqual(t, 0, len(applied.Env))
}

// --- vzCompletionProcess (the completionProcess adapter) ---

func TestDispatchVZ_CompletionProcess_Good_ExecStop(t *testing.T) {
	// The VM is already booted (spawnAgentVZ Runs synchronously); the adapter
	// drives only the Exec→Stop tail.
	fake := &fakeVZDispatcher{
		available:  true,
		execResult: core.Ok("agent stdout"),
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
}

func TestDispatchVZ_CompletionProcess_Ugly_ExecFails(t *testing.T) {
	// go-container folds a non-zero guest exit into a Fail error; the adapter
	// treats any exec failure as a failed run and still stops the VM.
	fake := &fakeVZDispatcher{
		available:  true,
		execResult: core.Fail(core.E("VZProvider.Exec", "command exited 2; stderr: boom", nil)),
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
	fake := &fakeVZDispatcher{available: true, execResult: core.Ok("vz output"), stopResult: core.Ok(nil)}
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
