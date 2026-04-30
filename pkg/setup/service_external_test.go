// SPDX-License-Identifier: EUPL-1.2

package setup_test

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
	"dappco.re/go/agent/pkg/setup"
)

func TestGitOrigin_Service_DetectGitRemote_Good(t *testing.T) {
	dir := t.TempDir()
	c := core.New()
	core.RequireTrue(t, agentic.ProcessRegister(c).OK)
	service := &setup.Service{ServiceRuntime: core.NewServiceRuntime(c, setup.RuntimeOptions{})}

	core.RequireTrue(t, c.Process().RunIn(context.Background(), dir, "git", "init").OK)
	core.RequireTrue(t, c.Process().RunIn(context.Background(), dir, "git", "remote", "add", "origin", "git@forge.lthn.ai:core/agent.git").OK)

	core.AssertEqual(t, "core/agent", service.DetectGitRemote(dir))
}
