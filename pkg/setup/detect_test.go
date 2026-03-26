// SPDX-License-Identifier: EUPL-1.2

package setup

import (
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
)

func TestDetect_Detect_Good_Go(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "go.mod"), "module test")
	assert.Equal(t, TypeGo, Detect(dir))
}

func TestDetect_Detect_Good_PHP(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "composer.json"), "{}")
	assert.Equal(t, TypePHP, Detect(dir))
}

func TestDetect_Detect_Bad_Unknown(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, TypeUnknown, Detect(dir))
}

func TestDetect_DetectAll_Good(t *testing.T) {
	dir := t.TempDir()
	fs.Write(core.JoinPath(dir, "go.mod"), "module test")
	fs.Write(core.JoinPath(dir, "package.json"), "{}")
	types := DetectAll(dir)
	assert.Contains(t, types, TypeGo)
	assert.Contains(t, types, TypeNode)
}

func TestDetect_AbsolutePath_Ugly_Empty(t *testing.T) {
	result := absolutePath("")
	assert.NotEmpty(t, result)
}
