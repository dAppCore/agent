// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleAPIError_Error() {
	err := &forgeAPIError{StatusCode: 404, Path: "/api/v1/repos/core/agent", Message: "not found"}
	core.Println(err.Error())
	// Output: forge /api/v1/repos/core/agent returned HTTP 404: not found
}
