// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestService_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	service := c.Service("setup")
	assert.True(t, service.OK)
	svc, ok := service.Value.(*Service)
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

func TestService_OnStartup_Good(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}

	result := service.OnStartup(context.Background())
	assert.True(t, result.OK)
}

func TestService_OnStartup_Bad_CancelledContext(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := service.OnStartup(ctx)
	assert.True(t, result.OK)
}

func TestService_OnStartup_Ugly_NilRuntime(t *testing.T) {
	result := (&Service{}).OnStartup(context.Background())
	assert.True(t, result.OK)
}

func TestService_DetectGitRemote_Bad_NonGitDir(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	remote := service.DetectGitRemote(t.TempDir())
	assert.Equal(t, "", remote)
}

func TestService_DetectGitRemote_Ugly_EmptyPath(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	assert.NotPanics(t, func() {
		service.DetectGitRemote("")
	})
}
