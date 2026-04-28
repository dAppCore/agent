// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestService_Register_Good(t *testing.T) {
	c := core.New(core.WithService(Register))
	service := c.Service("setup")
	core.AssertTrue(t, service.OK)
	svc, ok := service.Value.(*Service)
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, svc)
}

func TestNilCore_Register_Bad(t *testing.T) {
	result := Register(nil)
	core.AssertTrue(t, result.OK)
	svc, ok := result.Value.(*Service)
	core.RequireTrue(t, ok)
	core.AssertNil(t, svc.Core())
}

func TestFreshInstance_Register_Ugly(t *testing.T) {
	c := core.New()
	first := Register(c)
	second := Register(c)
	core.RequireTrue(t, first.OK)
	core.RequireTrue(t, second.OK)
	firstService, ok := first.Value.(*Service)
	core.RequireTrue(t, ok)
	secondService, ok := second.Value.(*Service)
	core.RequireTrue(t, ok)
	core.AssertTrue(t, firstService != secondService)
}

func TestStartup_Service_OnStartup_Good(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}

	result := service.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
}

func TestCancelledContext_Service_OnStartup_Bad(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := service.OnStartup(ctx)
	core.AssertTrue(t, result.OK)
}

func TestNilRuntime_Service_OnStartup_Ugly(t *testing.T) {
	result := (&Service{}).OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestNonGitDir_Service_DetectGitRemote_Bad(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	remote := service.DetectGitRemote(t.TempDir())
	core.AssertEqual(t, "", remote)
}

func TestEmptyPath_Service_DetectGitRemote_Ugly(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	core.AssertNotPanics(t, func() {
		service.DetectGitRemote("")
	})
}
