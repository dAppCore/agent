// SPDX-License-Identifier: EUPL-1.2

package main

import (
	"reflect"
	"testing"

	core "github.com/dappcore/go"
)

func assertIsType(t *testing.T, want, got any, msg ...string) {
	t.Helper()
	core.AssertEqual(t, reflect.TypeOf(want), reflect.TypeOf(got), msg...)
}
