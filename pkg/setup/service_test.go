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

func TestService_DetectGitRemote_Good_NonGitDir(t *testing.T) {
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{})}
	// Non-git dir returns empty
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
