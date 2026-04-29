// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/process"
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

func TestService_Register_Bad(t *testing.T) {
	result := Register(nil)
	core.AssertTrue(t, result.OK)
	svc, ok := result.Value.(*Service)
	core.RequireTrue(t, ok)
	core.AssertNil(t, svc.Core())
}

func TestService_Register_Ugly(t *testing.T) {
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

func TestService_Service_OnStartup_Good(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}

	result := service.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
}

func TestService_Service_OnStartup_Bad(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := service.OnStartup(ctx)
	core.AssertTrue(t, result.OK)
}

func TestService_Service_OnStartup_Ugly(t *testing.T) {
	result := (&Service{}).OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestService_Service_DetectGitRemote_Good(t *testing.T) {
	dir := t.TempDir()
	c := core.New()
	factory := process.NewService(process.Options{})
	instance, err := factory(c)
	core.RequireNoError(t, err)
	processService, ok := instance.(*process.Service)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, c.RegisterService("process", processService).OK)
	core.RequireTrue(t, c.ServiceStartup(context.Background(), nil).OK)
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}

	_, err = processService.RunWithOptions(context.Background(), process.RunOptions{Command: "git", Args: []string{"init"}, Dir: dir})
	core.RequireNoError(t, err)
	_, err = processService.RunWithOptions(context.Background(), process.RunOptions{Command: "git", Args: []string{"remote", "add", "origin", "git@forge.lthn.ai:core/agent.git"}, Dir: dir})
	core.RequireNoError(t, err)

	core.AssertEqual(t, "core/agent", service.DetectGitRemote(dir))
}

func TestService_Service_DetectGitRemote_Bad(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	remote := service.DetectGitRemote(t.TempDir())
	core.AssertEqual(t, "", remote)
}

func TestService_Service_DetectGitRemote_Ugly(t *testing.T) {
	c := core.New()
	service := &Service{ServiceRuntime: core.NewServiceRuntime(c, RuntimeOptions{})}
	core.AssertNotPanics(t, func() {
		service.DetectGitRemote("")
	})
}
