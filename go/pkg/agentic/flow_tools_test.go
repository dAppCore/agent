// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/agent/pkg/lib/flow"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// withFlowEnumerator swaps the per-flow tool enumerator for the duration of a
// test and restores it afterwards.
//
//	withFlowEnumerator(t, func() []flow.Flow { return []flow.Flow{{Name: "release"}} })
func withFlowEnumerator(t *testing.T, enumerator func() []flow.Flow) {
	t.Helper()
	previous := flowToolEnumerator
	flowToolEnumerator = enumerator
	t.Cleanup(func() { flowToolEnumerator = previous })
}

// listFlowTools connects an in-memory MCP client to the registered server and
// returns the advertised tools.
func listFlowTools(t *testing.T) []*mcpsdk.Tool {
	t.Helper()

	svc, err := coremcp.New(coremcp.Options{Unrestricted: true})
	core.RequireNoError(t, err)

	subsystem := &PrepSubsystem{}
	subsystem.RegisterTools(svc)

	server := svc.Server()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "0.1.0"}, nil)
	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = serverSession.Close() })

	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	core.RequireNoError(t, err)
	t.Cleanup(func() { _ = clientSession.Close() })

	result, err := clientSession.ListTools(context.Background(), nil)
	core.RequireNoError(t, err)
	return result.Tools
}

func TestFlowTools_RegisterFlowTools_Good_DeclaredFlowBecomesTool(t *testing.T) {
	withFlowEnumerator(t, func() []flow.Flow {
		return []flow.Flow{{
			Name:        "release",
			Description: "Cut a release",
			Inputs: []flow.Input{
				{Name: "version", Type: "string", Required: true, Description: "semver to tag"},
				{Name: "draft", Type: "bool", Required: false, Description: "create a draft"},
			},
		}}
	})

	var releaseTool *mcpsdk.Tool
	for _, tool := range listFlowTools(t) {
		if tool.Name == "agentic_flow_release" {
			releaseTool = tool
			break
		}
	}
	if releaseTool == nil {
		t.Fatal("agentic_flow_release tool was not registered")
	}
	if releaseTool.Description != "Cut a release" {
		t.Fatalf("description = %q, want %q", releaseTool.Description, "Cut a release")
	}
	if releaseTool.InputSchema == nil {
		t.Fatal("registered flow tool has no input schema")
	}
}

func TestFlowTools_flowInputSchema_Good_DerivesSchemaFromInputs(t *testing.T) {
	schema := flowInputSchema([]flow.Input{
		{Name: "version", Type: "string", Required: true, Description: "semver to tag"},
		{Name: "draft", Type: "bool", Required: false, Description: "create a draft"},
		{Name: "retries", Type: "int", Required: true},
	})

	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties has type %T, want map", schema["properties"])
	}
	version, ok := properties["version"].(map[string]any)
	if !ok {
		t.Fatalf("version property has type %T, want map", properties["version"])
	}
	if version["type"] != "string" {
		t.Fatalf("version type = %v, want string", version["type"])
	}
	if version["description"] != "semver to tag" {
		t.Fatalf("version description = %v, want %q", version["description"], "semver to tag")
	}
	draft, ok := properties["draft"].(map[string]any)
	if !ok {
		t.Fatalf("draft property has type %T, want map", properties["draft"])
	}
	if draft["type"] != "boolean" {
		t.Fatalf("draft type = %v, want boolean", draft["type"])
	}
	retries, ok := properties["retries"].(map[string]any)
	if !ok {
		t.Fatalf("retries property has type %T, want map", properties["retries"])
	}
	if retries["type"] != "integer" {
		t.Fatalf("retries type = %v, want integer", retries["type"])
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("required has type %T, want []string", schema["required"])
	}
	if len(required) != 2 {
		t.Fatalf("required = %v, want 2 entries", required)
	}
}

func TestFlowTools_RegisterFlowTools_Bad_UnnamedFlowSkipped(t *testing.T) {
	withFlowEnumerator(t, func() []flow.Flow {
		return []flow.Flow{
			{Name: "", Steps: []flow.Step{{Name: "x", Cmd: "y"}}},
			{Name: "keeper", Steps: []flow.Step{{Name: "x", Cmd: "y"}}},
		}
	})

	var names []string
	for _, tool := range listFlowTools(t) {
		names = append(names, tool.Name)
	}
	core.AssertContains(t, names, "agentic_flow_keeper")
	for _, name := range names {
		if name == "agentic_flow_" {
			t.Fatal("an unnamed flow was registered as a tool")
		}
	}
}

func TestFlowTools_RegisterFlowTools_Ugly_NoInputsStillRegisters(t *testing.T) {
	withFlowEnumerator(t, func() []flow.Flow {
		return []flow.Flow{{Name: "go-qa", Steps: []flow.Step{{Name: "build", Cmd: "go"}}}}
	})

	registered := false
	for _, candidate := range listFlowTools(t) {
		if candidate.Name == "agentic_flow_go_qa" {
			registered = true
			break
		}
	}
	if !registered {
		t.Fatal("agentic_flow_go_qa tool was not registered")
	}

	// A flow that declares no inputs still advertises an object schema with
	// empty properties and no required key.
	schema := flowInputSchema(nil)
	if schema["type"] != "object" {
		t.Fatalf("schema type = %v, want object", schema["type"])
	}
	if _, present := schema["required"]; present {
		t.Fatal("flow with no required inputs should omit the required key")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties has type %T, want map", schema["properties"])
	}
	if len(properties) != 0 {
		t.Fatalf("properties = %v, want empty", properties)
	}
}

func TestFlowTools_flowToolName_Good_SlugsNameToToolName(t *testing.T) {
	cases := map[string]string{
		"release":         "agentic_flow_release",
		"v0.8.0 Upgrade":  "agentic_flow_v0_8_0_upgrade",
		"Go QA  Pipeline": "agentic_flow_go_qa_pipeline",
	}
	for in, want := range cases {
		if got := flowToolName(in); got != want {
			t.Fatalf("flowToolName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFlowTools_flowInputJSONType_Good_MapsDeclaredTypes(t *testing.T) {
	cases := map[string]string{
		"int":     "integer",
		"bool":    "boolean",
		"string":  "string",
		"":        "string",
		"unknown": "string",
	}
	for declared, want := range cases {
		if got := flowInputJSONType(declared); got != want {
			t.Fatalf("flowInputJSONType(%q) = %q, want %q", declared, got, want)
		}
	}
}
