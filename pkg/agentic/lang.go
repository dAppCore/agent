// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var supportedLanguages = []string{"go", "php", "ts", "rust", "py", "cpp", "docker"}

// input := agentic.LanguageDetectInput{Path: "/workspace/pkg/agentic"}
type LanguageDetectInput struct {
	Path string `json:"path,omitempty"`
}

// out := agentic.LanguageDetectOutput{Success: true, Path: "/workspace/pkg/agentic", Language: "go"}
type LanguageDetectOutput struct {
	Success  bool   `json:"success"`
	Path     string `json:"path"`
	Language string `json:"language"`
}

// out := agentic.LanguageListOutput{Success: true, Count: 7, Languages: []string{"go", "php"}}
type LanguageListOutput struct {
	Success   bool     `json:"success"`
	Count     int      `json:"count"`
	Languages []string `json:"languages"`
}

type LanguageListInput struct{}

func (s *PrepSubsystem) registerLanguageCommands() {
	c := s.Core()
	c.Command("lang/detect", core.Command{Description: "Detect the primary language for a workspace or repository", Action: s.cmdLangDetect})
	c.Command("lang/list", core.Command{Description: "List supported language identifiers", Action: s.cmdLangList})
}

// result := c.Command("lang/detect").Run(ctx, core.NewOptions(core.Option{Key: "path", Value: "."}))
func (s *PrepSubsystem) cmdLangDetect(options core.Options) core.Result {
	path := optionStringValue(options, "_arg", "path", "repo")
	if path == "" {
		core.Print(nil, "usage: core-agent lang detect <path>")
		return core.Result{Value: core.E("agentic.cmdLangDetect", "path is required", nil), OK: false}
	}

	_, output, err := s.langDetect(context.Background(), nil, LanguageDetectInput{Path: path})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "path:     %s", output.Path)
	core.Print(nil, "language: %s", output.Language)
	return core.Result{Value: output, OK: true}
}

// result := c.Command("lang/list").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) cmdLangList(_ core.Options) core.Result {
	_, output, err := s.langList(context.Background(), nil, LanguageListInput{})
	if err != nil {
		core.Print(nil, "error: %v", err)
		return core.Result{Value: err, OK: false}
	}

	core.Print(nil, "%d language(s)", output.Count)
	for _, language := range output.Languages {
		core.Print(nil, "  %s", language)
	}
	return core.Result{Value: output, OK: true}
}

func (s *PrepSubsystem) registerLanguageTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "lang_detect",
		Description: "Detect the primary language for a workspace or repository path.",
	}, s.langDetect)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "lang_list",
		Description: "List supported language identifiers.",
	}, s.langList)
}

func (s *PrepSubsystem) langDetect(_ context.Context, _ *mcp.CallToolRequest, input LanguageDetectInput) (*mcp.CallToolResult, LanguageDetectOutput, error) {
	path := core.Trim(input.Path)
	if path == "" {
		path = "."
	}
	language := detectLanguage(path)
	return nil, LanguageDetectOutput{
		Success:  true,
		Path:     path,
		Language: language,
	}, nil
}

func (s *PrepSubsystem) langList(_ context.Context, _ *mcp.CallToolRequest, _ LanguageListInput) (*mcp.CallToolResult, LanguageListOutput, error) {
	languages := append([]string(nil), supportedLanguages...)
	return nil, LanguageListOutput{
		Success:   true,
		Count:     len(languages),
		Languages: languages,
	}, nil
}
