// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect_Detect_Good_Go(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	assert.Equal(t, TypeGo, Detect(dir))
}

func TestDetect_Detect_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "composer.json"), "{}", 0644).OK)
	assert.Equal(t, TypePHP, Detect(dir))
}

func TestDetect_Detect_Good_Node(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	assert.Equal(t, TypeNode, Detect(dir))
}

func TestDetect_Detect_Good_Wails(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	assert.Equal(t, TypeWails, Detect(dir))
}

func TestDetect_Detect_Bad_Unknown(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, TypeUnknown, Detect(dir))
}

func TestDetect_Detect_Ugly_WailsWinsOverGo(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	require.True(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	assert.Equal(t, TypeWails, Detect(dir))
}

func TestDetect_DetectAll_Good_Polyglot(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	require.True(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	types := DetectAll(dir)
	assert.Contains(t, types, TypeGo)
	assert.Contains(t, types, TypeNode)
	assert.NotContains(t, types, TypePHP)
}

func TestDetect_DetectAll_Bad_Empty(t *testing.T) {
	dir := t.TempDir()
	assert.Empty(t, DetectAll(dir))
}

func TestDetect_DetectAll_Ugly_StableOrder(t *testing.T) {
	dir := t.TempDir()
	require.True(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	require.True(t, fs.WriteMode(core.JoinPath(dir, "composer.json"), "{}", 0644).OK)
	require.True(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	require.True(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	assert.Equal(t, []ProjectType{TypeGo, TypePHP, TypeNode, TypeWails}, DetectAll(dir))
}

func TestDetect_AbsolutePath_Good_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, core.Path(dir), absolutePath(dir))
}

func TestDetect_AbsolutePath_Bad_EmptyUsesDirCWD(t *testing.T) {
	assert.Equal(t, core.Env("DIR_CWD"), absolutePath(""))
}

func TestDetect_AbsolutePath_Ugly_RelativeSegments(t *testing.T) {
	assert.Equal(t, core.Path("./repo/../repo"), absolutePath("./repo/../repo"))
}
