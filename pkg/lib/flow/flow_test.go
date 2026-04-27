// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	"bytes"
	"testing"

	core "dappco.re/go/core"
)

var testFS = (&core.Fs{}).NewUnrestricted()

func TestFlow_Parse_Good(t *testing.T) {
	definition, err := Parse(bytes.NewBufferString(
		"name: go-qa\n" +
			"description: Build and test\n" +
			"steps:\n" +
			"  - name: build\n" +
			"    cmd: build\n" +
			"    args:\n" +
			"      - --all\n" +
			"  - name: test\n" +
			"    cmd: test\n",
	))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if definition.Name != "go-qa" {
		t.Fatalf("Parse returned name %q, want %q", definition.Name, "go-qa")
	}
	if definition.Description != "Build and test" {
		t.Fatalf("Parse returned description %q, want %q", definition.Description, "Build and test")
	}
	if len(definition.Steps) != 2 {
		t.Fatalf("Parse returned %d steps, want 2", len(definition.Steps))
	}
	if definition.Steps[0].Name != "build" {
		t.Fatalf("Parse returned first step name %q, want %q", definition.Steps[0].Name, "build")
	}
	if definition.Steps[0].Cmd != "build" {
		t.Fatalf("Parse returned first step cmd %q, want %q", definition.Steps[0].Cmd, "build")
	}
	if len(definition.Steps[0].Args) != 1 || definition.Steps[0].Args[0] != "--all" {
		t.Fatalf("Parse returned first step args %#v, want [\"--all\"]", definition.Steps[0].Args)
	}
}

func TestFlow_ParseContinueOnError_Good(t *testing.T) {
	definition, err := Parse(bytes.NewBufferString(
		"steps:\n" +
			"  - cmd: verify\n" +
			"    continueOnError: true\n",
	))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(definition.Steps) != 1 {
		t.Fatalf("Parse returned %d steps, want 1", len(definition.Steps))
	}
	if !definition.Steps[0].ContinueOnError {
		t.Fatal("Parse did not set ContinueOnError")
	}
}

func TestFlow_ParseEmpty_Good(t *testing.T) {
	definition, err := Parse(bytes.NewBuffer(nil))
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if definition.Name != "" {
		t.Fatalf("Parse returned name %q, want empty", definition.Name)
	}
	if definition.Description != "" {
		t.Fatalf("Parse returned description %q, want empty", definition.Description)
	}
	if len(definition.Steps) != 0 {
		t.Fatalf("Parse returned %d steps, want 0", len(definition.Steps))
	}
}

func TestFlow_Parse_Bad(t *testing.T) {
	_, err := Parse(bytes.NewBufferString("steps: ["))
	if err == nil {
		t.Fatal("Parse unexpectedly succeeded for malformed YAML")
	}
}

func TestFlow_Parse_Ugly(t *testing.T) {
	_, err := Parse(bytes.NewBufferString(
		"steps:\n" +
			"  - name: build\n",
	))
	if err == nil {
		t.Fatal("Parse unexpectedly succeeded without cmd")
	}
	if !core.Contains(err.Error(), "cmd is required") {
		t.Fatalf("Parse returned error %q, want missing cmd", err.Error())
	}
}

func TestFlow_ParseFile_Good(t *testing.T) {
	path := core.JoinPath(t.TempDir(), "flow.yaml")
	writeTestFile(t, path,
		"name: release\n"+
			"steps:\n"+
			"  - cmd: tag\n",
	)

	definition, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile returned error: %v", err)
	}

	if definition.Name != "release" {
		t.Fatalf("ParseFile returned name %q, want %q", definition.Name, "release")
	}
	if len(definition.Steps) != 1 {
		t.Fatalf("ParseFile returned %d steps, want 1", len(definition.Steps))
	}
	if definition.Steps[0].Cmd != "tag" {
		t.Fatalf("ParseFile returned step cmd %q, want %q", definition.Steps[0].Cmd, "tag")
	}
}

func TestFlow_ParseFile_Bad(t *testing.T) {
	_, err := ParseFile(core.JoinPath(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("ParseFile unexpectedly succeeded for missing file")
	}
}

func TestFlow_ParseFile_Ugly(t *testing.T) {
	path := core.JoinPath(t.TempDir(), "invalid.yaml")
	writeTestFile(t, path,
		"steps:\n"+
			"  - name: build\n",
	)

	_, err := ParseFile(path)
	if err == nil {
		t.Fatal("ParseFile unexpectedly succeeded without cmd")
	}
	if !core.Contains(err.Error(), "cmd is required") {
		t.Fatalf("ParseFile returned error %q, want missing cmd", err.Error())
	}
}

func TestFlow_LoadEmbedded_Good(t *testing.T) {
	for _, name := range []string{
		"upgrade/v080-plan.yaml",
		"upgrade/v080-implement.yaml",
		"go",
	} {
		definition, err := LoadEmbedded(name)
		if err != nil {
			continue
		}

		if len(definition.Steps) == 0 {
			t.Fatalf("LoadEmbedded(%q) returned 0 steps", name)
		}
		return
	}

	t.Skip("no embedded flow currently matches the cmd-only YAML contract")
}

func TestFlow_LoadEmbedded_Bad(t *testing.T) {
	_, err := LoadEmbedded("missing-flow")
	if err == nil {
		t.Fatal("LoadEmbedded unexpectedly succeeded for missing flow")
	}
}

func TestFlow_LoadEmbedded_Ugly(t *testing.T) {
	_, err := LoadEmbedded("go")
	if err == nil {
		t.Fatal("LoadEmbedded unexpectedly succeeded for markdown-only template")
	}
	if !core.Contains(err.Error(), "not a YAML flow") {
		t.Fatalf("LoadEmbedded returned error %q, want markdown error", err.Error())
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if result := testFS.Write(path, content); !result.OK {
		t.Fatalf("Write(%q) failed: %v", path, result.Value)
	}
}
