// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	"bytes"
	"embed"
	"io"

	core "dappco.re/go/core"
	"gopkg.in/yaml.v3"
)

var fs = (&core.Fs{}).NewUnrestricted()

const parseFileContext = "flow.ParseFile"

//go:embed *.md upgrade
var embeddedFiles embed.FS

type Flow struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

type Step struct {
	Name            string   `yaml:"name"`
	Cmd             string   `yaml:"cmd"`
	Args            []string `yaml:"args"`
	ContinueOnError bool     `yaml:"continueOnError"`
}

func Parse(reader io.Reader) (Flow, error) {
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

func ParseFile(path string) (Flow, error) {
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

	return Parse(bytes.NewBufferString(content))
}

func LoadEmbedded(name string) (Flow, error) {
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
			return Parse(bytes.NewReader(frontMatter))
		}

		return Parse(bytes.NewReader(content))
	}

	return Flow{}, core.E("flow.LoadEmbedded", core.Concat("embedded flow not found: ", name), nil)
}

func validate(definition Flow) error {
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

func markdownFrontMatter(content []byte) ([]byte, bool) {
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return nil, false
	}

	content = content[len("---\n"):]
	index := bytes.Index(content, []byte("\n---\n"))
	if index < 0 {
		return nil, false
	}

	return content[:index], true
}
