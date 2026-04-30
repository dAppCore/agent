// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go"
)

func TestGo_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.AssertEqual(t, TypeGo, Detect(dir))
}

func TestPHP_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "composer.json"), "{}", 0644).OK)
	core.AssertEqual(t, TypePHP, Detect(dir))
}

func TestNode_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	core.AssertEqual(t, TypeNode, Detect(dir))
}

func TestWails_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	core.AssertEqual(t, TypeWails, Detect(dir))
}

func TestUnknown_Detect_Bad(t *testing.T) {
	dir := t.TempDir()
	got := Detect(dir)
	core.AssertEqual(t, TypeUnknown, got)
	core.AssertNotEqual(t, TypeGo, got)
}

func TestWailsWins_Detect_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	core.AssertEqual(t, TypeWails, Detect(dir))
}

func TestPolyglot_DetectAll_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	types := DetectAll(dir)
	core.AssertContains(t, types, TypeGo)
	core.AssertContains(t, types, TypeNode)
	core.AssertNotContains(t, types, TypePHP)
}

func TestEmpty_DetectAll_Bad(t *testing.T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	core.AssertEmpty(t, types)
	core.AssertLen(t, types, 0)
}

func TestStableOrder_DetectAll_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "composer.json"), "{}", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	core.AssertEqual(t, []ProjectType{TypeGo, TypePHP, TypeNode, TypeWails}, DetectAll(dir))
}

func TestDetect_AbsolutePath_Good_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	got := absolutePath(dir)
	core.AssertEqual(t, core.Path(dir), got)
	core.AssertNotEmpty(t, got)
}

func TestDetect_AbsolutePath_Bad_EmptyUsesDirCWD(t *testing.T) {
	got := absolutePath("")
	core.AssertEqual(t, core.Env("DIR_CWD"), got)
	core.AssertNotEmpty(t, got)
}

func TestDetect_AbsolutePath_Ugly_RelativeSegments(t *testing.T) {
	got := absolutePath("./repo/../repo")
	core.AssertEqual(t, core.Path("./repo/../repo"), got)
	core.AssertContains(t, got, "repo")
}

func TestDetect_Detect_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.AssertEqual(t, TypeGo, Detect(dir))
}

func TestDetect_Detect_Bad(t *testing.T) {
	dir := t.TempDir()
	got := Detect(dir)
	core.AssertEqual(t, TypeUnknown, got)
	core.AssertNotEqual(t, TypeGo, got)
}

func TestDetect_Detect_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	core.AssertEqual(t, TypeWails, Detect(dir))
}

func TestDetect_DetectAll_Good(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	types := DetectAll(dir)
	core.AssertContains(t, types, TypeGo)
	core.AssertContains(t, types, TypeNode)
	core.AssertNotContains(t, types, TypePHP)
}

func TestDetect_DetectAll_Bad(t *testing.T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	core.AssertEmpty(t, types)
	core.AssertLen(t, types, 0)
}

func TestDetect_DetectAll_Ugly(t *testing.T) {
	dir := t.TempDir()
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "go.mod"), "module test\n", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "composer.json"), "{}", 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "package.json"), `{"name":"test"}`, 0644).OK)
	core.RequireTrue(t, fs.WriteMode(core.JoinPath(dir, "wails.json"), `{}`, 0644).OK)
	core.AssertEqual(t, []ProjectType{TypeGo, TypePHP, TypeNode, TypeWails}, DetectAll(dir))
}
