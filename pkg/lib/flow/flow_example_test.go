// SPDX-License-Identifier: EUPL-1.2

package flow

import (
	core "dappco.re/go"
)

func ExampleParse() {
	definition, err := Parse(core.NewReader(`name: demo
steps:
  - name: build
    cmd: go
    args:
      - test
      - ./...
`))
	core.Println(err == nil)
	core.Println(definition.Name)
	// Output:
	// true
	// demo
}

func ExampleParseFile() {
	fsys := (&core.Fs{}).NewUnrestricted()
	path := "/tmp/core-agent-flow-example.yaml"
	defer fsys.DeleteAll(path)
	writeResult := fs.Write(path, `name: demo
steps:
  - name: build
    cmd: go
`)
	definition, err := ParseFile(path)
	core.Println(writeResult.OK)
	core.Println(err == nil)
	core.Println(definition.Name)
	// Output:
	// true
	// true
	// demo
}

func ExampleLoadEmbedded() {
	definition, err := LoadEmbedded("upgrade/v080-plan")
	core.Println(err != nil)
	core.Println(definition.Name == "")
	// Output:
	// true
	// true
}
