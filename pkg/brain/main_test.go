// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("CORE_BRAIN_INSECURE", "true")
	if os.Getenv("CORE_BRAIN_INSECURE") != "true" {
		os.Exit(1)
	}
	os.Exit(m.Run())
}
