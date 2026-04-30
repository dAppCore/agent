// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func ExampleRepoSyncOutput() {
	output := RepoSyncOutput{Repo: "core/go-io", Branch: "dev"}
	core.Println(output.Repo)
	// Output: core/go-io
}
