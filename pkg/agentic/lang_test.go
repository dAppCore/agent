// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"testing"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLang_CmdLangDetect_Good_Go(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module example").OK)

	r := s.cmdLangDetect(core.NewOptions(core.Option{Key: "_arg", Value: dir}))
	require.True(t, r.OK)

	output, ok := r.Value.(LanguageDetectOutput)
	require.True(t, ok)
	assert.Equal(t, dir, output.Path)
	assert.Equal(t, "go", output.Language)
}

func TestLang_CmdLangDetect_Bad_MissingPath(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	r := s.cmdLangDetect(core.NewOptions())
	assert.False(t, r.OK)
}

func TestLang_CmdLangDetect_Ugly_PreferenceOrder(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "go.mod"), "module example").OK)
	require.True(t, fs.Write(core.JoinPath(dir, "package.json"), "{}").OK)

	r := s.cmdLangDetect(core.NewOptions(core.Option{Key: "_arg", Value: dir}))
	require.True(t, r.OK)

	output, ok := r.Value.(LanguageDetectOutput)
	require.True(t, ok)
	assert.Equal(t, "go", output.Language)
}

func TestLang_LangDetect_Good_PHP(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)
	dir := t.TempDir()
	require.True(t, fs.Write(core.JoinPath(dir, "composer.json"), "{}").OK)

	_, output, err := s.langDetect(context.Background(), (*mcp.CallToolRequest)(nil), LanguageDetectInput{Path: dir})
	require.NoError(t, err)
	assert.Equal(t, dir, output.Path)
	assert.Equal(t, "php", output.Language)
}

func TestLang_LangList_Good(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, output, err := s.langList(context.Background(), (*mcp.CallToolRequest)(nil), LanguageListInput{})
	require.NoError(t, err)
	require.True(t, output.Success)
	assert.Equal(t, len(supportedLanguages), output.Count)
	assert.Equal(t, supportedLanguages, output.Languages)
}

func TestLang_LangList_Ugly_CopyIsolation(t *testing.T) {
	s, _ := testPrepWithCore(t, nil)

	_, first, err := s.langList(context.Background(), (*mcp.CallToolRequest)(nil), LanguageListInput{})
	require.NoError(t, err)
	require.NotEmpty(t, first.Languages)
	first.Languages[0] = "mutated"

	_, second, err := s.langList(context.Background(), (*mcp.CallToolRequest)(nil), LanguageListInput{})
	require.NoError(t, err)
	assert.Equal(t, supportedLanguages[0], second.Languages[0])
}
