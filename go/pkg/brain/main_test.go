// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"syscall"
	"testing"

	core "dappco.re/go"
)

func TestMain(m *testing.M) {
	_ = syscall.Setenv("CORE_BRAIN_INSECURE", "true")
	if value, ok := syscall.Getenv("CORE_BRAIN_INSECURE"); !ok || value != "true" {
		core.Exit(1)
	}
	core.Exit(m.Run())
}
