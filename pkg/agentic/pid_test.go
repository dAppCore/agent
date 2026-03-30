// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- PIDAlive ---

func TestPid_PIDAlive_Good(t *testing.T) {
	pid, _ := strconv.Atoi(core.Env("PID"))
	assert.True(t, PIDAlive(pid))
}

func TestPid_PIDAlive_Bad(t *testing.T) {
	assert.False(t, PIDAlive(999999))
}

func TestPid_PIDAlive_Ugly(t *testing.T) {
	assert.False(t, PIDAlive(0))
}

// --- PIDTerminate ---

func TestPid_PIDTerminate_Good(t *testing.T) {
	r := testCore.Process().Start(context.Background(), core.NewOptions(
		core.Option{Key: "command", Value: "sleep"},
		core.Option{Key: "args", Value: []string{"30"}},
		core.Option{Key: "detach", Value: true},
	))
	require.True(t, r.OK)

	proc, ok := r.Value.(*process.Process)
	require.True(t, ok)

	pid := proc.Info().PID
	require.NotZero(t, pid)

	defer func() {
		_ = proc.Kill()
	}()

	assert.True(t, PIDTerminate(pid))

	select {
	case <-proc.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("PIDTerminate did not stop the process")
	}

	assert.False(t, PIDAlive(pid))
}

func TestPid_PIDTerminate_Bad(t *testing.T) {
	assert.False(t, PIDTerminate(999999))
}

func TestPid_PIDTerminate_Ugly(t *testing.T) {
	assert.False(t, PIDTerminate(0))
}
