// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"testing"
	"time"

	core "dappco.re/go"
)

// testPrep is the package-level PrepSubsystem for tests that need process execution.
var testPrep *PrepSubsystem

// testCore is the package-level Core with go-process registered.
var testCore *core.Core

// TestMain sets up a PrepSubsystem with go-process registered for all tests in the package.
func TestMain(m *testing.M) {
	testRoot, err := os.MkdirTemp("", "agentic-tests-*")
	if err != nil {
		panic(err)
	}
	homeDir := core.JoinPath(testRoot, "home")
	_ = os.MkdirAll(homeDir, 0o755)
	_ = os.MkdirAll(core.JoinPath(homeDir, "Code", ".core"), 0o755)

	_ = os.Setenv("CORE_BRAIN_INSECURE", "true")
	_ = os.Setenv("CORE_HOME", homeDir)
	_ = os.Setenv("HOME", homeDir)
	_ = os.Setenv("DIR_HOME", homeDir)
	testCore = core.New(
		core.WithService(ProcessRegister),
	)
	testCore.ServiceStartup(context.Background(), nil)

	// Enable pipeline feature flags (matches Register defaults)
	testCore.Config().Enable("auto-qa")
	testCore.Config().Enable("auto-pr")
	testCore.Config().Enable("auto-merge")
	testCore.Config().Enable("auto-ingest")

	testPrep = &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
	os.Exit(m.Run())
}

// newPrepWithProcess creates a PrepSubsystem wired to testCore for tests that
// need process execution via s.Core().Process().
func newPrepWithProcess() *PrepSubsystem {
	return &PrepSubsystem{
		ServiceRuntime: core.NewServiceRuntime(testCore, AgentOptions{}),
		backoff:        make(map[string]time.Time),
		failCount:      make(map[string]int),
	}
}

// --- ProcessAlive ---

func TestPid_ProcessAlive_Good(t *testing.T) {
	proc := startManagedProcess(t, testCore)
	pid := proc.Info().PID

	core.AssertTrue(t, ProcessAlive(testCore, proc.ID, pid))
	core.AssertTrue(t, ProcessAlive(testCore, "", pid))
}

func TestPid_ProcessAlive_Bad(t *testing.T) {
	alive := ProcessAlive(testCore, "", 999999)
	core.AssertFalse(t, alive)
	core.AssertEqual(t, false, alive)
}

func TestPid_ProcessAlive_Ugly(t *testing.T) {
	alive := ProcessAlive(nil, "", 0)
	core.AssertFalse(t, alive)
	core.AssertEqual(t, false, alive)
}

// --- ProcessTerminate ---

func TestPid_ProcessTerminate_Good(t *testing.T) {
	proc := startManagedProcess(t, testCore)
	pid := proc.Info().PID

	core.AssertTrue(t, ProcessTerminate(testCore, proc.ID, pid))

	select {
	case <-proc.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessTerminate did not stop the process")
	}

	core.AssertFalse(t, ProcessAlive(testCore, proc.ID, pid))
}

func TestPid_ProcessTerminate_Bad(t *testing.T) {
	terminated := ProcessTerminate(testCore, "", 999999)
	core.AssertFalse(t, terminated)
	core.AssertEqual(t, false, terminated)
}

func TestPid_ProcessTerminate_Ugly(t *testing.T) {
	terminated := ProcessTerminate(nil, "", 0)
	core.AssertFalse(t, terminated)
	core.AssertEqual(t, false, terminated)
}
