// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestLang_CmdLangDetect_Good_Go(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module example").OK)

	r := s.cmdLangDetect(core.NewOptions(core.Option{Key: "_arg", Value: dir}))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(LanguageDetectOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, dir, output.Path)
	core.AssertEqual(t, "go", output.Language)
}

func TestLang_CmdLangDetect_Bad_MissingPath(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdLangDetect(core.NewOptions())
	core.AssertFalse(t, r.OK)
}

func TestLang_CmdLangDetect_Ugly_PreferenceOrder(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "go.mod"), "module example").OK)
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)

	r := s.cmdLangDetect(core.NewOptions(core.Option{Key: "_arg", Value: dir}))
	core.RequireTrue(t, r.OK)

	output, ok := r.Value.(LanguageDetectOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, "go", output.Language)
}

func TestLang_LangDetect_Good_PHP(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	core.RequireTrue(t, fs.Write(core.JoinPath(dir, "composer.json"), "{}").OK)

	result := s.langDetect(context.Background(), LanguageDetectInput{Path: dir})
	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(LanguageDetectOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, dir, output.Path)
	core.AssertEqual(t, "php", output.Language)
}

func TestLang_LangList_Good_Case(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	result := s.langList(context.Background(), LanguageListInput{})
	core.RequireTrue(t, result.OK)
	output, ok := result.Value.(LanguageListOutput)
	core.RequireTrue(t, ok)
	core.RequireTrue(t, output.Success)
	core.AssertEqual(t, len(supportedLanguages), output.Count)
	core.AssertEqual(t, supportedLanguages, output.Languages)
}

func TestLang_LangList_Ugly_CopyIsolation(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	firstResult := s.langList(context.Background(), LanguageListInput{})
	core.RequireTrue(t, firstResult.OK)
	first, ok := firstResult.Value.(LanguageListOutput)
	core.RequireTrue(t, ok)
	core.RequireNotEmpty(t, first.Languages)
	first.Languages[0] = "mutated"

	secondResult := s.langList(context.Background(), LanguageListInput{})
	core.RequireTrue(t, secondResult.OK)
	second, ok := secondResult.Value.(LanguageListOutput)
	core.RequireTrue(t, ok)
	core.AssertEqual(t, supportedLanguages[0], second.Languages[0])
}
