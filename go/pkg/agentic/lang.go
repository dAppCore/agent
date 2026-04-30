// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var supportedLanguages = []string{"go", "php", "ts", "rust", "py", "cpp", "docker"}

// input := agentic.LanguageDetectInput{Path: "/workspace/pkg/agentic"}
type LanguageDetectInput struct {
	Path string "json:\"path,omitempty\""
}

// out := agentic.LanguageDetectOutput{Success: true, Path: "/workspace/pkg/agentic", Language: "go"}
type LanguageDetectOutput struct {
	Success  bool   `json:"success"`
	Path     string "json:\"path\""
	Language string `json:"language"`
}

// out := agentic.LanguageListOutput{Success: true, Count: 7, Languages: []string{"go", "php"}}
type LanguageListOutput struct {
	Success   bool     `json:"success"`
	Count     int      `json:"count"`
	Languages []string `json:"languages"`
}

type LanguageListInput struct{}

func (s *PrepSubsystem) registerLanguageCommands() core.Result {
	c := s.Core()
	if r := c.Command("lang/detect", core.Command{Description: "Detect the primary language for a workspace or repository", Action: s.cmdLangDetect}); !r.OK {
		return r
	}
	if r := c.Command("agentic:lang/detect", core.Command{Description: "Detect the primary language for a workspace or repository", Action: s.cmdLangDetect}); !r.OK {
		return r
	}
	if r := c.Command("lang/list", core.Command{Description: "List supported language identifiers", Action: s.cmdLangList}); !r.OK {
		return r
	}
	if r := c.Command("agentic:lang/list", core.Command{Description: "List supported language identifiers", Action: s.cmdLangList}); !r.OK {
		return r
	}
	return core.Ok(nil)
}

// result := c.Command("lang/detect").Run(ctx, core.NewOptions(core.Option{Key: `path`, Value: "."}))
func (s *PrepSubsystem) cmdLangDetect(options core.Options) core.Result {
	path := optionStringValue(options, "_arg", `path`, "repo")
	if path == "" {
		core.Print(nil, "usage: core-agent lang detect <path>")
		return core.Result{Value: core.E("agentic.cmdLangDetect", "path is required", nil), OK: false}
	}

	result := typedResultValue[LanguageDetectOutput]("lang.detect", "invalid language detect output", s.langDetect(context.Background(), LanguageDetectInput{Path: path}))
	if !result.OK {
		core.Print(nil, "error: %s", result.Error())
		return result
	}
	output, _ := result.Value.(LanguageDetectOutput)

	core.Print(nil, "path:     %s", output.Path)
	core.Print(nil, "language: %s", output.Language)
	return core.Ok(output)
}

// result := c.Command("lang/list").Run(ctx, core.NewOptions())
func (s *PrepSubsystem) cmdLangList(_ core.Options) core.Result {
	result := typedResultValue[LanguageListOutput]("lang.list", "invalid language list output", s.langList(context.Background(), LanguageListInput{}))
	if !result.OK {
		core.Print(nil, "error: %s", result.Error())
		return result
	}
	output, _ := result.Value.(LanguageListOutput)

	core.Print(nil, "%d language(s)", output.Count)
	for _, language := range output.Languages {
		core.Print(nil, "  %s", language)
	}
	return core.Ok(output)
}

func (s *PrepSubsystem) registerLanguageTools(svc *coremcp.Service) {
	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "lang_detect",
		Description: "Detect the primary language for a workspace or repository path.",
	}, toolHandlerFor[LanguageDetectInput, LanguageDetectOutput]("lang.detect", "invalid language detect output", s.langDetect))

	coremcp.AddToolRecorded(svc, svc.Server(), "agentic", &mcp.Tool{
		Name:        "lang_list",
		Description: "List supported language identifiers.",
	}, toolHandlerFor[LanguageListInput, LanguageListOutput]("lang.list", "invalid language list output", s.langList))
}

func (s *PrepSubsystem) langDetect(_ context.Context, input LanguageDetectInput) core.Result {
	path := core.Trim(input.Path)
	if path == "" {
		path = "."
	}
	language := detectLanguage(path)
	return core.Ok(LanguageDetectOutput{
		Success:  true,
		Path:     path,
		Language: language,
	})
}

func (s *PrepSubsystem) langList(_ context.Context, _ LanguageListInput) core.Result {
	languages := append([]string(nil), supportedLanguages...)
	return core.Ok(LanguageListOutput{
		Success:   true,
		Count:     len(languages),
		Languages: languages,
	})
}
