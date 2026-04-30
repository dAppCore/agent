// SPDX-License-Identifier: EUPL-1.2

package monitor

import core "dappco.re/go"

func ExampleCheckinResponse() {
	resp := CheckinResponse{Timestamp: 1712345678}
	core.Println(resp.Timestamp > 0)
	// Output: true
}
