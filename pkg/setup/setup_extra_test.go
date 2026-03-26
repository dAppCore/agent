// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- defaultBuildCommand ---

func TestSetup_DefaultBuildCommand_Good(t *testing.T) {
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeGo))
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeWails))
	assert.Equal(t, "composer test", defaultBuildCommand(TypePHP))
	assert.Equal(t, "npm run build", defaultBuildCommand(TypeNode))
	assert.Equal(t, "make build", defaultBuildCommand(TypeUnknown))
}

// --- defaultTestCommand ---

func TestSetup_DefaultTestCommand_Good(t *testing.T) {
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeGo))
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeWails))
	assert.Equal(t, "composer test", defaultTestCommand(TypePHP))
	assert.Equal(t, "npm test", defaultTestCommand(TypeNode))
	assert.Equal(t, "make test", defaultTestCommand(TypeUnknown))
}

// --- formatFlow ---

func TestSetup_FormatFlow_Good(t *testing.T) {
	goFlow := formatFlow(TypeGo)
	assert.Contains(t, goFlow, "go build ./...")
	assert.Contains(t, goFlow, "go test ./...")

	phpFlow := formatFlow(TypePHP)
	assert.Contains(t, phpFlow, "composer test")

	nodeFlow := formatFlow(TypeNode)
	assert.Contains(t, nodeFlow, "npm run build")
	assert.Contains(t, nodeFlow, "npm test")
}

// --- Detect ---

func TestSetup_DetectGo_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/go.mod", "module test\n")
	assert.Equal(t, TypeGo, Detect(dir))
}

func TestSetup_DetectPHP_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/composer.json", `{"name":"test"}`)
	assert.Equal(t, TypePHP, Detect(dir))
}

func TestSetup_DetectNode_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/package.json", `{"name":"test"}`)
	assert.Equal(t, TypeNode, Detect(dir))
}

func TestSetup_DetectWails_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/wails.json", `{}`)
	assert.Equal(t, TypeWails, Detect(dir))
}

func TestSetup_Detect_Bad(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, TypeUnknown, Detect(dir))
}

// --- DetectAll ---

func TestSetup_DetectAll_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/go.mod", "module test\n")
	fs.Write(dir+"/package.json", `{"name":"test"}`)

	types := DetectAll(dir)
	assert.Contains(t, types, TypeGo)
	assert.Contains(t, types, TypeNode)
	assert.NotContains(t, types, TypePHP)
}

func TestSetup_DetectAll_Bad(t *testing.T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	assert.Empty(t, types)
}
