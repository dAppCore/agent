// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"context"

	core "dappco.re/go"
)

// options := setup.SetupOptions{}
type SetupOptions struct{}

// RuntimeOptions is kept as a compatibility alias for older callers.
type RuntimeOptions = SetupOptions

// c := core.New(core.WithService(setup.Register))
// service := c.Service("setup")
type Service struct {
	*core.ServiceRuntime[SetupOptions]
}

// c := core.New(core.WithService(setup.Register))
// service := c.Service("setup")
func Register(c *core.Core) core.Result {
	service := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{}),
	}
	return core.Result{Value: service, OK: true}
}

// result := service.OnStartup(context.Background())
func (s *Service) OnStartup(ctx context.Context) core.Result {
	return core.Result{OK: true}
}

// remote := service.DetectGitRemote("/srv/repos/agent")
func (s *Service) DetectGitRemote(path string) string {
	result := s.Core().Process().RunIn(context.Background(), path, "git", "remote", "get-url", "origin")
	if !result.OK {
		return ""
	}
	return parseGitRemote(core.Trim(result.Value.(string)))
}
