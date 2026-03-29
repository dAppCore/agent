// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	svc, ok := core.ServiceFor[*Service](c, "setup")
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

func TestService_OnStartup_Good(t *testing.T) {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}

	result := svc.OnStartup(context.Background())
	assert.True(t, result.OK)
}

func TestService_OnStartup_Bad_CancelledContext(t *testing.T) {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := svc.OnStartup(ctx)
	assert.True(t, result.OK)
}

func TestService_OnStartup_Ugly_NilRuntime(t *testing.T) {
	result := (&Service{}).OnStartup(context.Background())
	assert.True(t, result.OK)
}

func TestService_DetectGitRemote_Good_GitOrigin(t *testing.T) {
	dir := t.TempDir()
	c := core.New()
	require.True(t, agentic.ProcessRegister(c).OK)
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}

	require.True(t, c.Process().RunIn(context.Background(), dir, "git", "init").OK)
	require.True(t, c.Process().RunIn(context.Background(), dir, "git", "remote", "add", "origin", "git@forge.lthn.ai:core/agent.git").OK)

	assert.Equal(t, "core/agent", svc.DetectGitRemote(dir))
}

func TestService_DetectGitRemote_Bad_NonGitDir(t *testing.T) {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
	remote := svc.DetectGitRemote(t.TempDir())
	assert.Equal(t, "", remote)
}

func TestService_DetectGitRemote_Ugly_EmptyPath(t *testing.T) {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
	assert.NotPanics(t, func() {
		svc.DetectGitRemote("")
	})
}
