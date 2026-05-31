// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	iofs "io/fs"

	core "dappco.re/go"
)

// ListEmbedded returns every embedded flow that parses into a valid Flow,
// ordered by embed path. Files that are not structured YAML flows (prose
// markdown without front matter, or step shapes that fail validation) are
// skipped, so the result is exactly the set of flows a runner — or the MCP
// tool registrar — can act on. Each returned Flow carries its declared
// Inputs, which is the schema source for per-flow MCP tool registration.
//
//	for _, f := range flow.ListEmbedded() {
//		core.Println(f.Name, len(f.Inputs))
//	}
func ListEmbedded() []Flow {
	var flows []Flow
	_ = iofs.WalkDir(embeddedFiles, ".", func(path string, entry iofs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if !hasFlowExtension(path) {
			return nil
		}
		definition, loadErr := LoadEmbedded(path)
		if loadErr != nil {
			return nil
		}
		if core.Trim(definition.Name) == "" && len(definition.Steps) == 0 {
			return nil
		}
		flows = append(flows, definition)
		return nil
	})
	return flows
}
