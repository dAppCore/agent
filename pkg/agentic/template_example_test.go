// SPDX-License-Identifier: EUPL-1.2

package agentic

import core "dappco.re/go"

func Example_templateVariableList() {
	variables := templateVariableList(planTemplateDefinition{
		Variables: map[string]planTemplateVariableDef{
			"feature_name": {Required: true},
			"description":  {Required: false},
		},
	})

	core.Println(variables[0].Name, variables[0].Required)
	core.Println(variables[1].Name, variables[1].Required)
	// Output:
	// description false
	// feature_name true
}
