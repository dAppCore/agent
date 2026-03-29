// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"os"
	"testing"

	"dappco.re/go/agent/pkg/agentic"
	core "dappco.re/go/core"
)

var testMon *Subsystem

func TestMain(m *testing.M) {
	c := core.New(core.WithService(agentic.ProcessRegister))
	c.ServiceStartup(context.Background(), nil)
	testMon = New()
	testMon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	os.Exit(m.Run())
}
