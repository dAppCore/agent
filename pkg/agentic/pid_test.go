// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"testing"
	"time"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

// testPrep is the package-level PrepSubsystem for tests that need process execution.
var testPrep *PrepSubsystem

// testCore is the package-level Core with go-process registered.
var testCore *core.Core

// TestMain sets up a PrepSubsystem with go-process registered for all tests in the package.
func TestMain(m *testing.M) {
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

	assert.True(t, ProcessAlive(testCore, proc.ID, pid))
	assert.True(t, ProcessAlive(testCore, "", pid))
}

func TestPid_ProcessAlive_Bad(t *testing.T) {
	assert.False(t, ProcessAlive(testCore, "", 999999))
}

func TestPid_ProcessAlive_Ugly(t *testing.T) {
	assert.False(t, ProcessAlive(nil, "", 0))
}

// --- ProcessTerminate ---

func TestPid_ProcessTerminate_Good(t *testing.T) {
	proc := startManagedProcess(t, testCore)
	pid := proc.Info().PID

	assert.True(t, ProcessTerminate(testCore, proc.ID, pid))

	select {
	case <-proc.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("ProcessTerminate did not stop the process")
	}

	assert.False(t, ProcessAlive(testCore, proc.ID, pid))
}

func TestPid_ProcessTerminate_Bad(t *testing.T) {
	assert.False(t, ProcessTerminate(testCore, "", 999999))
}

func TestPid_ProcessTerminate_Ugly(t *testing.T) {
	assert.False(t, ProcessTerminate(nil, "", 0))
}
