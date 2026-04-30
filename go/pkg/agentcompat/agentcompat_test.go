// SPDX-License-Identifier: EUPL-1.2

package agentcompat

import (
	"testing"

	core "dappco.re/go"
	"gopkg.in/yaml.v3"
)

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAMLResult_Good(t *testing.T) {
	var node yaml.Node
	core.RequireNoError(t, yaml.Unmarshal([]byte("total: 4\ngpt-5.4: 2\n"), &node))
	var limit ConcurrencyLimit
	result := limit.UnmarshalYAMLResult(node.Content[0])
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 4, limit.Total)
	core.AssertEqual(t, 2, limit.Models["gpt-5.4"])
}

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAMLResult_Bad(t *testing.T) {
	var node yaml.Node
	core.RequireNoError(t, yaml.Unmarshal([]byte("[not, a, limit]\n"), &node))
	var limit ConcurrencyLimit
	result := limit.UnmarshalYAMLResult(node.Content[0])
	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAMLResult_Ugly(t *testing.T) {
	var node yaml.Node
	core.RequireNoError(t, yaml.Unmarshal([]byte("0\n"), &node))
	limit := ConcurrencyLimit{Models: map[string]int{"stale": 9}}
	result := limit.UnmarshalYAMLResult(node.Content[0])
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, 0, limit.Total)
	core.AssertEqual(t, 9, limit.Models["stale"])
}

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAML_Good(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("3\n"), &limit)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 3, limit.Total)
}

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAML_Bad(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("[bad]\n"), &limit)
	core.AssertError(t, err)
}

func TestAgentcompat_ConcurrencyLimit_UnmarshalYAML_Ugly(t *testing.T) {
	var limit ConcurrencyLimit
	err := yaml.Unmarshal([]byte("total: 1\ncodex: 0\n"), &limit)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 0, limit.Models["codex"])
}

func TestAgentcompat_HTTPStream_SendResult_Good(t *testing.T) {
	server := core.NewHTTPTestServer(core.HandlerFunc(func(w core.ResponseWriter, r *core.Request) {
		core.WriteString(w, `{"ok":true}`)
	}))
	defer server.Close()
	stream := &HTTPStream{Client: server.Client(), URL: server.URL, Method: "POST"}
	result := stream.SendResult([]byte(`{"ping":1}`))
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, `{"ok":true}`, string(stream.Response))
}

func TestAgentcompat_HTTPStream_SendResult_Bad(t *testing.T) {
	stream := &HTTPStream{URL: "http://example.invalid", Method: "POST"}
	result := stream.SendResult([]byte(`{"ping":1}`))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "client is required")
}

func TestAgentcompat_HTTPStream_SendResult_Ugly(t *testing.T) {
	var stream *HTTPStream
	result := stream.SendResult([]byte(`{"ping":1}`))
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "stream is required")
}

func TestAgentcompat_HTTPStream_Send_Good(t *testing.T) {
	server := core.NewHTTPTestServer(core.HandlerFunc(func(w core.ResponseWriter, r *core.Request) {
		core.WriteString(w, `{"pong":1}`)
	}))
	defer server.Close()
	stream := &HTTPStream{Client: server.Client(), URL: server.URL, Method: "POST"}
	err := stream.Send([]byte(`{"ping":1}`))
	core.AssertNoError(t, err)
	core.AssertEqual(t, `{"pong":1}`, string(stream.Response))
}

func TestAgentcompat_HTTPStream_Send_Bad(t *testing.T) {
	stream := &HTTPStream{Client: &core.HTTPClient{}, URL: "://bad", Method: "POST"}
	err := stream.Send([]byte(`{"ping":1}`))
	core.AssertError(t, err)
}

func TestAgentcompat_HTTPStream_Send_Ugly(t *testing.T) {
	var stream *HTTPStream
	core.AssertPanics(t, func() {
		stream.Send([]byte(`{"ping":1}`))
	})
}

func TestAgentcompat_HTTPStream_ReceiveResult_Good(t *testing.T) {
	stream := &HTTPStream{Response: []byte(`{"cached":true}`)}
	result := stream.ReceiveResult()
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, []byte(`{"cached":true}`), result.Value)
}

func TestAgentcompat_HTTPStream_ReceiveResult_Bad(t *testing.T) {
	var stream *HTTPStream
	result := stream.ReceiveResult()
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "stream is required")
}

func TestAgentcompat_HTTPStream_ReceiveResult_Ugly(t *testing.T) {
	stream := &HTTPStream{}
	result := stream.ReceiveResult()
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, []byte(nil), result.Value)
}

func TestAgentcompat_HTTPStream_Receive_Good(t *testing.T) {
	stream := &HTTPStream{Response: []byte("ok")}
	data, err := stream.Receive()
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte("ok"), data)
}

func TestAgentcompat_HTTPStream_Receive_Bad(t *testing.T) {
	var stream *HTTPStream
	result := stream.ReceiveResult()
	core.AssertFalse(t, result.OK)
	core.AssertContains(t, result.Error(), "stream is required")
}

func TestAgentcompat_HTTPStream_Receive_Ugly(t *testing.T) {
	stream := &HTTPStream{}
	data, err := stream.Receive()
	core.AssertNoError(t, err)
	core.AssertEqual(t, []byte(nil), data)
}

func TestAgentcompat_HTTPStream_CloseResult_Good(t *testing.T) {
	stream := &HTTPStream{}
	result := stream.CloseResult()
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, result.Value)
}

func TestAgentcompat_HTTPStream_CloseResult_Bad(t *testing.T) {
	var stream *HTTPStream
	result := stream.CloseResult()
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, stream)
}

func TestAgentcompat_HTTPStream_CloseResult_Ugly(t *testing.T) {
	stream := &HTTPStream{Response: []byte("stale")}
	result := stream.CloseResult()
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, []byte("stale"), stream.Response)
}

func TestAgentcompat_HTTPStream_Close_Good(t *testing.T) {
	stream := &HTTPStream{}
	err := stream.Close()
	core.AssertNoError(t, err)
}

func TestAgentcompat_HTTPStream_Close_Bad(t *testing.T) {
	var stream *HTTPStream
	err := stream.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, stream)
}

func TestAgentcompat_HTTPStream_Close_Ugly(t *testing.T) {
	stream := &HTTPStream{Token: "token"}
	err := stream.Close()
	core.AssertNoError(t, err)
	core.AssertEqual(t, "token", stream.Token)
}
