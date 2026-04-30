// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"

	core "dappco.re/go"
)

func TestForgeClient_APIError_Error_Good(t *testing.T) {
	err := &forgeAPIError{StatusCode: 404, Path: "/api/v1/repos/core/agent", Message: "not found"}
	core.AssertContains(t, err.Error(), "not found")
	core.AssertContains(t, err.Error(), "404")
}

func TestForgeClient_APIError_Error_Bad(t *testing.T) {
	var err *forgeAPIError
	message := err.Error()
	core.AssertEqual(t, "forge API error", message)
	core.AssertNotContains(t, message, "HTTP")
}

func TestForgeClient_APIError_Error_Ugly(t *testing.T) {
	err := &forgeAPIError{StatusCode: 500, Path: "/api/v1/repos/core/agent"}
	core.AssertContains(t, err.Error(), "HTTP 500")
	core.AssertNotContains(t, err.Error(), "not found")
}
