// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- defaultBuildCommand ---

func TestDefaultBuildCommand_Good_Go(t *testing.T) {
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeGo))
}

func TestDefaultBuildCommand_Good_Wails(t *testing.T) {
	assert.Equal(t, "go build ./...", defaultBuildCommand(TypeWails))
}

func TestDefaultBuildCommand_Good_PHP(t *testing.T) {
	assert.Equal(t, "composer test", defaultBuildCommand(TypePHP))
}

func TestDefaultBuildCommand_Good_Node(t *testing.T) {
	assert.Equal(t, "npm run build", defaultBuildCommand(TypeNode))
}

func TestDefaultBuildCommand_Good_Unknown(t *testing.T) {
	assert.Equal(t, "make build", defaultBuildCommand(TypeUnknown))
}

// --- defaultTestCommand ---

func TestDefaultTestCommand_Good_Go(t *testing.T) {
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeGo))
}

func TestDefaultTestCommand_Good_Wails(t *testing.T) {
	assert.Equal(t, "go test ./...", defaultTestCommand(TypeWails))
}

func TestDefaultTestCommand_Good_PHP(t *testing.T) {
	assert.Equal(t, "composer test", defaultTestCommand(TypePHP))
}

func TestDefaultTestCommand_Good_Node(t *testing.T) {
	assert.Equal(t, "npm test", defaultTestCommand(TypeNode))
}

func TestDefaultTestCommand_Good_Unknown(t *testing.T) {
	assert.Equal(t, "make test", defaultTestCommand(TypeUnknown))
}

// --- formatFlow ---

func TestFormatFlow_Good_Go(t *testing.T) {
	result := formatFlow(TypeGo)
	assert.Contains(t, result, "go build ./...")
	assert.Contains(t, result, "go test ./...")
}

func TestFormatFlow_Good_PHP(t *testing.T) {
	result := formatFlow(TypePHP)
	assert.Contains(t, result, "composer test")
}

func TestFormatFlow_Good_Node(t *testing.T) {
	result := formatFlow(TypeNode)
	assert.Contains(t, result, "npm run build")
	assert.Contains(t, result, "npm test")
}

// --- Detect ---

func TestDetect_Good_GoProject(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/go.mod", "module test\n")
	assert.Equal(t, TypeGo, Detect(dir))
}

func TestDetect_Good_PHPProject(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/composer.json", `{"name":"test"}`)
	assert.Equal(t, TypePHP, Detect(dir))
}

func TestDetect_Good_NodeProject(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/package.json", `{"name":"test"}`)
	assert.Equal(t, TypeNode, Detect(dir))
}

func TestDetect_Good_WailsProject(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/wails.json", `{}`)
	assert.Equal(t, TypeWails, Detect(dir))
}

func TestDetect_Good_Unknown(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, TypeUnknown, Detect(dir))
}

// --- DetectAll ---

func TestDetectAll_Good_Polyglot(t *testing.T) {
	dir := t.TempDir()
	fs.Write(dir+"/go.mod", "module test\n")
	fs.Write(dir+"/package.json", `{"name":"test"}`)

	types := DetectAll(dir)
	assert.Contains(t, types, TypeGo)
	assert.Contains(t, types, TypeNode)
	assert.NotContains(t, types, TypePHP)
}

func TestDetectAll_Good_Empty(t *testing.T) {
	dir := t.TempDir()
	types := DetectAll(dir)
	assert.Empty(t, types)
}
