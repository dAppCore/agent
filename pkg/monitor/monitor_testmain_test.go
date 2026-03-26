// SPDX-License-Identifier: EUPL-1.2

package monitor

import (
	"context"
	"os"
	"testing"

	core "dappco.re/go/core"
	"dappco.re/go/core/process"
)

var testMon *Subsystem

func TestMain(m *testing.M) {
	c := core.New(core.WithService(process.Register))
	c.ServiceStartup(context.Background(), nil)
	testMon = New()
	testMon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})
	os.Exit(m.Run())
}
