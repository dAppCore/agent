// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineTraining_Good_RootHelp(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	output := captureStdout(t, func() {
		result := s.cmdPipelineTraining(core.NewOptions())
		require.True(t, result.OK)
	})

	assert.Contains(t, output, "core-agent pipeline/training/capture")
	assert.Contains(t, output, "core-agent pipeline/training/stats")
	assert.Contains(t, output, "core-agent pipeline/training/export")
}

func TestPipelineTraining_Bad_CaptureBlockedOnSibling(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.cmdPipelineTrainingCapture(core.NewOptions())

	require.False(t, result.OK)
	err, ok := result.Value.(error)
	require.True(t, ok)
	assert.Contains(t, err.Error(), "blocked-on-sibling")
}
