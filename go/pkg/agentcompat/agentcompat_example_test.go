// SPDX-License-Identifier: EUPL-1.2

package agentcompat

import (
	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

func ExampleConcurrencyLimit_UnmarshalYAMLResult() {
	var node yaml.Node
	yaml.Unmarshal([]byte("total: 2\ncodex: 1\n"), &node)
	limit := ConcurrencyLimit{}
	result := limit.UnmarshalYAMLResult(node.Content[0])
	core.Println(result.OK, limit.Total, limit.Models["codex"])
	// Output: true 2 1
}

func ExampleConcurrencyLimit_UnmarshalYAML() {
	limit := ConcurrencyLimit{}
	yaml.Unmarshal([]byte("3\n"), &limit)
	core.Println(limit.Total)
	// Output: 3
}

func ExampleHTTPStream_SendResult() {
	stream := &HTTPStream{}
	result := stream.SendResult([]byte(`{"ping":1}`))
	core.Println(result.OK)
	// Output: false
}

func ExampleHTTPStream_Send() {
	server := core.NewHTTPTestServer(core.HandlerFunc(func(w core.ResponseWriter, r *core.Request) {
		core.WriteString(w, "ok")
	}))
	defer server.Close()
	stream := &HTTPStream{Client: server.Client(), URL: server.URL, Method: "POST"}
	err := stream.Send([]byte(`{"ping":1}`))
	core.Println(err == nil, string(stream.Response))
	// Output: true ok
}

func ExampleHTTPStream_ReceiveResult() {
	stream := &HTTPStream{Response: []byte("ok")}
	result := stream.ReceiveResult()
	core.Println(string(result.Value.([]byte)))
	// Output: ok
}

func ExampleHTTPStream_Receive() {
	stream := &HTTPStream{Response: []byte("ok")}
	data, err := stream.Receive()
	core.Println(string(data), err == nil)
	// Output: ok true
}

func ExampleHTTPStream_CloseResult() {
	stream := &HTTPStream{}
	result := stream.CloseResult()
	core.Println(result.OK)
	// Output: true
}

func ExampleHTTPStream_Close() {
	stream := &HTTPStream{}
	err := stream.Close()
	core.Println(err == nil)
	// Output: true
}
