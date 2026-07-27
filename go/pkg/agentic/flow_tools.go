// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib/flow"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// flowToolEnumerator yields the flows that are registered as individual MCP
// tools. It defaults to the embedded structured-flow set; tests override it to
// inject a flow with a known Inputs schema and assert the generated tool shape.
//
//	flowToolEnumerator = func() []flow.Flow { return []flow.Flow{{Name: "release"}} }
var flowToolEnumerator = flow.ListEmbedded

// flowToolInput is the argument map an enumerated flow tool accepts: declared
// input name → supplied value. The per-flow InputSchema (built from the flow's
// declared Inputs) is what a tool-using model reads; this map carries whatever
// the model sends back.
//
//	input := flowToolInput{"version": "1.2.0"}
type flowToolInput map[string]string

// FlowToolOutput reports the flow a per-flow MCP tool resolved and the args it
// validated against the flow's declared schema.
//
//	out := agentic.FlowToolOutput{Flow: "release", Valid: true}
type FlowToolOutput struct {
	Flow  string            `json:"flow"`
	Valid bool              `json:"valid"`
	Args  map[string]string `json:"args,omitempty"`
}

// registerFlowTools registers each enumerated flow as its own MCP tool whose
// InputSchema is generated from the flow's declared Inputs (Mantis #1804), so a
// tool-using model sees every flow as a callable tool with typed inputs.
//
//	subsystem.registerFlowTools(svc)
func (s *PrepSubsystem) registerFlowTools(svc *coremcp.Service) {
	if svc == nil {
		return
	}
	for _, definition := range flowToolEnumerator() {
		name := core.Trim(definition.Name)
		if name == "" {
			continue
		}
		registerFlowTool(svc, definition)
	}
}

// registerFlowTool registers a single flow as an MCP tool. Pulled out so the
// captured flow definition is per-iteration, not shared across the loop.
//
//	registerFlowTool(svc, flow.Flow{Name: "release"})
func registerFlowTool(svc *coremcp.Service, definition flow.Flow) {
	tool := &mcp.Tool{
		Name:        flowToolName(definition.Name),
		Description: flowToolDescription(definition),
		InputSchema: flowInputSchema(definition.Inputs),
	}
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", tool,
		func(_ context.Context, _ *mcp.CallToolRequest, input flowToolInput) (*mcp.CallToolResult, FlowToolOutput, error) {
			args := map[string]string(input)
			if err := definition.ValidateInputs(args); err != nil {
				return nil, FlowToolOutput{}, err
			}
			return nil, FlowToolOutput{Flow: definition.Name, Valid: true, Args: args}, nil
		})
}

// flowToolName maps a flow name to its MCP tool name, mirroring the
// `agentic_<flow>` shape the other agentic tools use.
//
//	flowToolName("v0.8.0 Upgrade") // "agentic_flow_v0_8_0_upgrade"
func flowToolName(flowName string) string {
	slug := core.Lower(core.Trim(flowName))
	cleaned := core.NewBuilder()
	previousUnderscore := false
	for _, r := range slug {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			cleaned.WriteRune(r)
			previousUnderscore = false
		default:
			if !previousUnderscore {
				cleaned.WriteRune('_')
				previousUnderscore = true
			}
		}
	}
	return core.Concat("agentic_flow_", core.TrimCutset(cleaned.String(), "_"))
}

// flowToolDescription builds the tool description from the flow's own
// description, falling back to a generic line when the flow declares none.
//
//	flowToolDescription(flow.Flow{Name: "release", Description: "Cut a release"})
func flowToolDescription(definition flow.Flow) string {
	if description := core.Trim(definition.Description); description != "" {
		return description
	}
	return core.Concat("Run the ", definition.Name, " flow.")
}

// flowInputSchema builds a JSON Schema object from a flow's declared Inputs so
// the registered MCP tool advertises typed, optionally-required parameters.
//
//	schema := flowInputSchema([]flow.Input{{Name: "version", Type: "string", Required: true}})
func flowInputSchema(inputs []flow.Input) map[string]any {
	properties := map[string]any{}
	var required []string
	for _, input := range inputs {
		name := core.Trim(input.Name)
		if name == "" {
			continue
		}
		property := map[string]any{"type": flowInputJSONType(input.Type)}
		if description := core.Trim(input.Description); description != "" {
			property["description"] = description
		}
		properties[name] = property
		if input.Required {
			required = append(required, name)
		}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// flowInputJSONType maps a flow input's declared type to its JSON Schema type.
// An empty or unknown type falls back to "string", mirroring the flow
// package's own default.
//
//	flowInputJSONType("int") // "integer"
func flowInputJSONType(declared string) string {
	switch core.Trim(declared) {
	case "int":
		return "integer"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}
