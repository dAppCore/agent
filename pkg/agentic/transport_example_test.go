// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
)

func ExampleRegisterHTTPTransport() {
	c := core.New()
	RegisterHTTPTransport(c)

	protocols := c.API().Protocols()
	core.Println(len(protocols))
	core.Println(protocols[0])
	core.Println(protocols[1])
	// Output:
	// 2
	// http
	// https
}

func ExampleStream_Send() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	stream := &httpStream{client: srv.Client(), url: srv.URL, token: "token-123", method: http.MethodPost}
	err := stream.Send([]byte(`{"ping":1}`))
	core.Println(err == nil)
	core.Println(string(stream.response))
	// Output:
	// true
	// {"ok":true}
}

func ExampleStream_Receive() {
	stream := &httpStream{response: []byte(`{"result":"pong"}`)}
	response, _ := stream.Receive()
	core.Println(string(response))
	// Output: {"result":"pong"}
}

func ExampleStream_Close() {
	stream := &httpStream{}
	core.Println(stream.Close() == nil)
	// Output: true
}

func ExampleHTTPGet() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"method":"GET"}`))
	}))
	defer srv.Close()

	result := HTTPGet(context.Background(), srv.URL, "token-123", "token")
	core.Println(result.Value.(string))
	// Output: {"method":"GET"}
}

func ExampleHTTPPost() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"method":"POST"}`))
	}))
	defer srv.Close()

	result := HTTPPost(context.Background(), srv.URL, `{"title":"demo"}`, "token-123", "token")
	core.Println(result.Value.(string))
	// Output: {"method":"POST"}
}

func ExampleHTTPPatch() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"method":"PATCH"}`))
	}))
	defer srv.Close()

	result := HTTPPatch(context.Background(), srv.URL, `{"title":"demo"}`, "token-123", "token")
	core.Println(result.Value.(string))
	// Output: {"method":"PATCH"}
}

func ExampleHTTPDelete() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"method":"DELETE"}`))
	}))
	defer srv.Close()

	result := HTTPDelete(context.Background(), srv.URL, `{"delete":true}`, "token-123", "Bearer")
	core.Println(result.Value.(string))
	// Output: {"method":"DELETE"}
}

func ExampleHTTPDo() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"method":"PUT"}`))
	}))
	defer srv.Close()

	result := HTTPDo(context.Background(), http.MethodPut, srv.URL, `{"title":"demo"}`, "token-123", "token")
	core.Println(result.Value.(string))
	// Output: {"method":"PUT"}
}

func ExampleDriveGet() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"repo":"go-io"}`))
	}))
	defer srv.Close()

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveGet(c, "forge", "/repos/core/go-io", "Bearer")
	core.Println(result.Value.(string))
	// Output: {"repo":"go-io"}
}

func ExampleDrivePost() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	defer srv.Close()

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "brain"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "brain-key"},
	))

	result := DrivePost(c, "brain", "/v1/brain/recall", `{"query":"build"}`, "Bearer")
	core.Println(result.Value.(string))
	// Output: {"created":true}
}

func ExampleDriveDo() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	defer srv.Close()

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveDo(c, "forge", http.MethodPatch, "/repos/core/go-io/pulls/5", `{"title":"AX"}`, "Bearer")
	core.Println(result.Value.(string))
	// Output: {"updated":true}
}
