// SPDX-License-Identifier: EUPL-1.2

package brain

import core "github.com/dappcore/go"

func ExampleRouteDescription() {
	route := RouteDescription{Method: "GET", Path: "/status", Summary: "Status"}
	core.Println(route.Method, route.Path)
	// Output: GET /status
}
