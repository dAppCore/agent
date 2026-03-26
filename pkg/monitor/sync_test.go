// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSync_InitSyncTimestamp_Good(t *testing.T) {
	mon := New()
	mon.initSyncTimestamp()
	assert.True(t, mon.lastSyncTimestamp > 0)
}

func TestSync_InitSyncTimestamp_Bad_NoOverwrite(t *testing.T) {
	mon := New()
	mon.lastSyncTimestamp = 42
	mon.initSyncTimestamp()
	assert.Equal(t, int64(42), mon.lastSyncTimestamp)
}

func TestSync_SyncRepos_Ugly_NoBrainKey(t *testing.T) {
	t.Setenv("CORE_BRAIN_KEY", "")
	mon := New()
	mon.ServiceRuntime = testMon.ServiceRuntime
	result := mon.syncRepos()
	assert.Equal(t, "", result)
}
