// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	core "dappco.re/go"
)

func TestProviderContract_RouteDescription_Good(t *testing.T) {
	route := RouteDescription{Method: "GET", Path: "/status", Summary: "Status"}
	core.AssertEqual(t, "GET", route.Method)
	core.AssertEqual(t, "/status", route.Path)
}

func TestProviderContract_RouteDescription_Bad(t *testing.T) {
	route := RouteDescription{}
	core.AssertEqual(t, "", route.Method)
	core.AssertEqual(t, "", route.Path)
}

func TestProviderContract_RouteDescription_Ugly(t *testing.T) {
	route := RouteDescription{Tags: []string{"brain"}, RequestBody: map[string]any{"query": "agent"}}
	core.AssertEqual(t, []string{"brain"}, route.Tags)
	core.AssertNotNil(t, route.RequestBody)
}
