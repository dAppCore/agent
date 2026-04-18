// SPDX-License-Identifier: EUPL-1.2

package setup_test

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/setup"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_DetectGitRemote_Good_GitOrigin(t *testing.T) {
	dir := t.TempDir()
	c := core.New()
	require.True(t, agentic.ProcessRegister(c).OK)
	service := &setup.Service{ServiceRuntime: core.NewServiceRuntime(c, setup.RuntimeOptions{})}

	require.True(t, c.Process().RunIn(context.Background(), dir, "git", "init").OK)
	require.True(t, c.Process().RunIn(context.Background(), dir, "git", "remote", "add", "origin", "git@forge.lthn.ai:core/agent.git").OK)

	assert.Equal(t, "core/agent", service.DetectGitRemote(dir))
}
