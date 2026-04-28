// SPDX-License-Identifier: EUPL-1.2

package agentic_test

import (
	core "dappco.re/go"
	"dappco.re/go/agent/pkg/agentic"
)

func ExampleRegisterHTTPTransport() {
	c := core.New()
	agentic.RegisterHTTPTransport(c)

	// HTTP and HTTPS protocols are now registered with Core API.
	core.Println(c.API().Protocols())
	// Output: [http https]
}

func ExampleDriveGet() {
	c := core.New()

	// Register a Drive endpoint
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: "https://forge.lthn.ai"},
		core.Option{Key: "token", Value: "my-token"},
	))

	// DriveGet reads base URL + token from the Drive handle.
	// r := agentic.DriveGet(c, "forge", "/api/v1/repos/core/go-io", "token")
	// if r.OK { body := r.Value.(string) }

	// Verify Drive is registered
	core.Println(c.Drive().Has("forge"))
	// Output: true
}

func ExampleDrivePost() {
	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "brain"},
		core.Option{Key: "transport", Value: "https://api.lthn.sh"},
		core.Option{Key: "token", Value: "brain-key"},
	))

	// DrivePost reads base URL + token from the Drive handle.
	// r := agentic.DrivePost(c, "brain", "/v1/brain/recall", body, "Bearer")

	core.Println(c.Drive().Has("brain"))
	// Output: true
}
