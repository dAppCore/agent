// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	"embed"
	"io"

	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

var fs = (&core.Fs{}).NewUnrestricted()

const parseFileContext = "flow.ParseFile"

//go:embed *.md upgrade
var embeddedFiles embed.FS

// Flow is the top-level YAML-defined workflow: a name, a description, and an
// ordered list of Steps that runners execute in sequence. Loaded via Parse,
// ParseFile, or LoadEmbedded.
//
//	flow, _ := flow.Parse(reader)
//	for _, step := range flow.Steps { /* run step */ }
type Flow struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

// Step is a single command invocation inside a Flow: the step name, the
// command + args to run, and whether failure should continue the flow or
// abort it.
//
//	step := flow.Step{Name: "build", Cmd: "task", Args: []string{"build"}}
type Step struct {
	Name            string   `yaml:"name"`
	Cmd             string   `yaml:"cmd"`
	Args            []string `yaml:"args"`
	ContinueOnError bool     `yaml:"continueOnError"`
}

// Parse decodes a Flow definition from a YAML stream and validates the
// shape (non-empty name, at least one step, valid step names). Returns
// the parsed Flow and a wrapped error on decode/validation failure.
//
//	flow, err := flow.Parse(bytes.NewBufferString(yamlSrc))
var Parse = func(reader io.Reader) (Flow, error) {
	if reader == nil {
		return Flow{}, core.E("flow.Parse", "reader is nil", nil)
	}

	decoder := yaml.NewDecoder(reader)

	var definition Flow
	if err := decoder.Decode(&definition); err != nil {
		if err == io.EOF {
			return Flow{}, nil
		}
		return Flow{}, core.E("flow.Parse", "decode flow YAML", err)
	}

	if err := validate(definition); err != nil {
		return Flow{}, err
	}

	return definition, nil
}

// ParseFile reads path via the unrestricted core filesystem and forwards the
// content to Parse. Useful for loading a flow definition from disk in one
// call. Returns a wrapped error on read failure or invalid YAML.
//
//	flow, err := flow.ParseFile(".core/flows/build.yaml")
var ParseFile = func(path string) (Flow, error) {
	readResult := fs.Read(path)
	if !readResult.OK {
		if err, ok := readResult.Value.(error); ok {
			return Flow{}, core.E(parseFileContext, core.Concat("read ", path), err)
		}
		return Flow{}, core.E(parseFileContext, core.Concat("read ", path), nil)
	}

	content, ok := readResult.Value.(string)
	if !ok {
		return Flow{}, core.E(parseFileContext, core.Concat("read ", path), nil)
	}

	return Parse(core.NewBufferString(content))
}

// LoadEmbedded loads a Flow definition baked into the binary via go:embed.
// Accepts a bare name and tries common candidate paths/extensions. Rejects
// path-escape attempts (leading slash, "..") and returns a wrapped error
// when the name is missing or no candidate matches.
//
//	flow, err := flow.LoadEmbedded("upgrade")
var LoadEmbedded = func(name string) (Flow, error) {
	name = normaliseEmbeddedName(name)
	if name == "" {
		return Flow{}, core.E("flow.LoadEmbedded", "name is required", nil)
	}

	if core.HasPrefix(name, "/") || core.Contains(name, "..") {
		return Flow{}, core.E("flow.LoadEmbedded", core.Concat("invalid embedded flow name: ", name), nil)
	}

	for _, candidate := range embeddedCandidates(name) {
		content, err := embeddedFiles.ReadFile(candidate)
		if err != nil {
			continue
		}

		if isMarkdown(candidate) {
			frontMatter, ok := markdownFrontMatter(content)
			if !ok {
				return Flow{}, core.E("flow.LoadEmbedded", core.Concat("embedded markdown is not a YAML flow: ", candidate), nil)
			}
			return Parse(core.NewReader(frontMatter))
		}

		return Parse(core.NewReader(string(content)))
	}

	return Flow{}, core.E("flow.LoadEmbedded", core.Concat("embedded flow not found: ", name), nil)
}

var validate = func(definition Flow) error {
	for index, step := range definition.Steps {
		if core.Trim(step.Cmd) != "" {
			continue
		}

		name := core.Trim(step.Name)
		if name == "" {
			name = core.Concat("step-", core.Sprintf("%d", index+1))
		}

		return core.E("flow.validate", core.Concat("step \"", name, "\" cmd is required"), nil)
	}

	return nil
}

func normaliseEmbeddedName(name string) string {
	name = core.Trim(name)
	name = core.TrimPrefix(name, "./")
	name = core.TrimPrefix(name, "pkg/lib/flow/")
	name = core.TrimPrefix(name, "flow/")
	return name
}

func embeddedCandidates(name string) []string {
	if hasFlowExtension(name) {
		return []string{name}
	}

	return []string{
		name + ".yaml",
		name + ".yml",
		name + ".md",
	}
}

func hasFlowExtension(name string) bool {
	return core.HasSuffix(name, ".yaml") || core.HasSuffix(name, ".yml") || core.HasSuffix(name, ".md")
}

func isMarkdown(name string) bool {
	return core.HasSuffix(name, ".md")
}

func markdownFrontMatter(content []byte) (string, bool) {
	text := core.Replace(string(content), "\r\n", "\n")
	if !core.HasPrefix(text, "---\n") {
		return "", false
	}

	text = text[len("---\n"):]
	parts := core.SplitN(text, "\n---\n", 2)
	if len(parts) != 2 {
		return "", false
	}

	return parts[0], true
}
