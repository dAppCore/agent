// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
)

func TestTransport_RegisterHTTPTransport_Good(t *testing.T) {
	c := core.New()

	RegisterHTTPTransport(c)

	core.AssertContains(t, c.API().Protocols(), "http")
	core.AssertContains(t, c.API().Protocols(), "https")
}

func TestTransport_HTTPGet_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "GET", r.Method)
		core.AssertEqual(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	result := HTTPGet(context.Background(), srv.URL, "test-token", "token")

	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"status":"ok"}`, result.Value.(string))
}

func TestInvalidURL_HTTPGet_Bad(t *testing.T) {
	result := HTTPGet(context.Background(), "://bad", "", "")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_DriveGet_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/repos/core/go-io", r.URL.Path)
		core.AssertEqual(t, "Bearer drive-token", r.Header.Get("Authorization"))
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

	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"repo":"go-io"}`, result.Value.(string))
}

func TestMissingDrive_DriveDo_Bad(t *testing.T) {
	result := DriveDo(core.New(), "missing", "PATCH", "/repos/core/go-io", `{"title":"AX"}`, "token")

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestServerError_HTTPDelete_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer srv.Close()

	result := HTTPDelete(context.Background(), srv.URL, "", "", "")

	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"upstream"}`, result.Value.(string))
}

func TestStreamRoundTrip_RegisterHTTPTransport_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "token api-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"echo":"ok"}`))
	}))
	defer srv.Close()

	c := core.New()
	RegisterHTTPTransport(c)
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "remote"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "api-token"},
	))

	streamResult := c.API().Stream("remote")
	core.RequireTrue(t, streamResult.OK)

	stream := streamResult.Value.(core.Stream)
	core.RequireNoError(t, stream.Send([]byte(`{"ping":1}`)))

	response, err := stream.Receive()
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"echo":"ok"}`, string(response))
	core.AssertNoError(t, stream.Close())
}

func TestTransport_Stream_Send_Bad(t *testing.T) {
	stream := &httpStream{client: defaultClient, url: "://bad", method: http.MethodPost}
	sendErr := stream.Send([]byte(`{"ping":1}`))

	core.AssertError(t, sendErr)
	core.AssertContains(t, sendErr.Error(), "missing protocol scheme")
}

func TestTransport_Stream_Send_Ugly(t *testing.T) {
	stream := &httpStream{url: "http://example.com", method: http.MethodPost}
	core.AssertPanics(t, func() {
		_ = stream.Send([]byte(`{"ping":1}`))
	})
}

func TestTransport_Stream_Receive_Bad(t *testing.T) {
	stream := &httpStream{}
	response, err := stream.Receive()

	core.RequireNoError(t, err)
	core.AssertNil(t, response)
}

func TestTransport_Stream_Receive_Ugly(t *testing.T) {
	var stream *httpStream
	core.AssertPanics(t, func() {
		_, _ = stream.Receive()
	})
}

func TestTransport_Stream_Close_Bad(t *testing.T) {
	stream := &httpStream{}
	err := stream.Close()

	core.RequireNoError(t, err)
	core.AssertNil(t, stream.client)
}

func TestTransport_Stream_Close_Ugly(t *testing.T) {
	var stream *httpStream
	err := stream.Close()
	core.AssertNoError(t, err)
	core.AssertNil(t, stream)
}

func TestTransport_RegisterHTTPTransport_Bad(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEmpty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	RegisterHTTPTransport(c)
	RegisterHTTPTransport(c)
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "remote"},
		core.Option{Key: "transport", Value: srv.URL},
	))

	streamResult := c.API().Stream("remote")
	core.RequireTrue(t, streamResult.OK)

	stream := streamResult.Value.(core.Stream)
	sendErr := stream.Send([]byte(`{"ping":1}`))
	core.RequireNoError(t, sendErr)

	response, receiveErr := stream.Receive()
	core.RequireNoError(t, receiveErr)
	core.AssertEqual(t, `{"status":"ok"}`, string(response))
}

func TestTransport_RegisterHTTPTransport_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "POST", r.Method)
		core.AssertEqual(t, "token api-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"echo":"ok"}`))
	}))
	defer srv.Close()

	c := core.New()
	RegisterHTTPTransport(c)
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "remote"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "api-token"},
	))

	streamResult := c.API().Stream("remote")
	core.RequireTrue(t, streamResult.OK)

	stream := streamResult.Value.(core.Stream)
	core.RequireNoError(t, stream.Send([]byte(`{"ping":1}`)))

	response, err := stream.Receive()
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"echo":"ok"}`, string(response))
	core.AssertNoError(t, stream.Close())
}

func TestTransport_HTTPGet_Bad(t *testing.T) {
	result := HTTPGet(context.Background(), "://bad", "", "")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_HTTPGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"get upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPGet(context.Background(), srv.URL, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"get upstream"}`, result.Value.(string))
}

func TestTransport_HTTPPost_Bad(t *testing.T) {
	result := HTTPPost(context.Background(), "://bad", `{"title":"Fix tests"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_HTTPPost_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"post upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPost(context.Background(), srv.URL, `{"title":"Fix tests"}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"post upstream"}`, result.Value.(string))
}

func TestTransport_HTTPPatch_Bad(t *testing.T) {
	result := HTTPPatch(context.Background(), "://bad", `{"status":"done"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_HTTPPatch_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"patch upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPatch(context.Background(), srv.URL, `{"status":"done"}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"patch upstream"}`, result.Value.(string))
}

func TestTransport_HTTPDelete_Bad(t *testing.T) {
	result := HTTPDelete(context.Background(), "://bad", `{"reason":"stale"}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_HTTPDelete_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer srv.Close()

	result := HTTPDelete(context.Background(), srv.URL, "", "", "")

	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"upstream"}`, result.Value.(string))
}

func TestTransport_HTTPDo_Bad(t *testing.T) {
	result := HTTPDo(context.Background(), http.MethodPut, "://bad", `{"value":7}`, "", "")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "create request")
}

func TestTransport_HTTPDo_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		_, _ = w.Write([]byte(`{"error":"put upstream"}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDo(context.Background(), http.MethodPut, srv.URL, `{"value":7}`, "", "")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"put upstream"}`, result.Value.(string))
}

func TestTransport_DriveGet_Bad(t *testing.T) {
	c := core.New()
	result := DriveGet(c, "missing", "/repos/core/go-io", "Bearer")

	core.AssertFalse(t, result.OK)
	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "drive not found")
}

func TestTransport_DriveGet_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"drive get upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveGet(c, "forge", "/repos/core/go-io", "Bearer")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive get upstream"}`, result.Value.(string))
}

func TestTransport_DrivePost_Bad(t *testing.T) {
	result := DrivePost(core.New(), "missing", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.AssertFalse(t, result.OK)

	err, ok := result.Value.(error)
	core.RequireTrue(t, ok)
	core.AssertContains(t, err.Error(), "drive not found")
}

func TestTransport_DrivePost_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"drive post upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DrivePost(c, "forge", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive post upstream"}`, result.Value.(string))
}

func TestTransport_DriveDo_Bad(t *testing.T) {
	result := DriveDo(core.New(), "missing", "PATCH", "/repos/core/go-io", `{"title":"AX"}`, "token")

	core.AssertFalse(t, result.OK)
	core.AssertError(t, result.Value.(error))
}

func TestTransport_DriveDo_Ugly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"drive do upstream"}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveDo(c, "forge", http.MethodPatch, "/pulls/3", `{"state":"closed"}`, "token")
	core.AssertFalse(t, result.OK)
	core.AssertEqual(t, `{"error":"drive do upstream"}`, result.Value.(string))
}

func TestTransport_Stream_Send_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "token send-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"ping":1}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"pong":1}`))
	}))
	t.Cleanup(srv.Close)

	stream := &httpStream{client: defaultClient, url: srv.URL, token: "send-token", method: http.MethodPost}
	sendErr := stream.Send([]byte(`{"ping":1}`))
	core.RequireNoError(t, sendErr)
	core.AssertEqual(t, `{"pong":1}`, string(stream.response))
}

func TestTransport_Stream_Receive_Good(t *testing.T) {
	stream := &httpStream{response: []byte(`{"cached":true}`)}
	response, err := stream.Receive()

	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"cached":true}`, string(response))
}

func TestTransport_Stream_Close_Good(t *testing.T) {
	stream := &httpStream{}
	err := stream.Close()

	core.RequireNoError(t, err)
	core.AssertNil(t, stream.response)
}

func TestTransport_HTTPPost_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer post-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"title":"Fix tests"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"created":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPost(context.Background(), srv.URL, `{"title":"Fix tests"}`, "post-token", "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"created":true}`, result.Value.(string))
}

func TestTransport_HTTPPatch_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "token patch-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"status":"done"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"updated":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPPatch(context.Background(), srv.URL, `{"status":"done"}`, "patch-token", "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"updated":true}`, result.Value.(string))
}

func TestTransport_HTTPDelete_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodDelete, r.Method)
		core.AssertEqual(t, "Bearer delete-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"reason":"stale"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"deleted":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDelete(context.Background(), srv.URL, `{"reason":"stale"}`, "delete-token", "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"deleted":true}`, result.Value.(string))
}

func TestTransport_HTTPDo_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, http.MethodPut, r.Method)
		core.AssertEqual(t, "token do-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"value":7}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)

	result := HTTPDo(context.Background(), http.MethodPut, srv.URL, `{"value":7}`, "do-token", "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"ok":true}`, result.Value.(string))
}

func TestTransport_DrivePost_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/issues", r.URL.Path)
		core.AssertEqual(t, http.MethodPost, r.Method)
		core.AssertEqual(t, "Bearer drive-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"title":"Follow up"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"number":9}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DrivePost(c, "forge", "/issues", `{"title":"Follow up"}`, "Bearer")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"number":9}`, result.Value.(string))
}

func TestTransport_DriveDo_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		core.AssertEqual(t, "/pulls/3", r.URL.Path)
		core.AssertEqual(t, http.MethodPatch, r.Method)
		core.AssertEqual(t, "token drive-token", r.Header.Get("Authorization"))

		bodyResult := core.ReadAll(r.Body)
		core.RequireTrue(t, bodyResult.OK)
		core.AssertEqual(t, `{"state":"closed"}`, bodyResult.Value.(string))
		_, _ = w.Write([]byte(`{"closed":true}`))
	}))
	t.Cleanup(srv.Close)

	c := core.New()
	c.Drive().New(core.NewOptions(
		core.Option{Key: "name", Value: "forge"},
		core.Option{Key: "transport", Value: srv.URL},
		core.Option{Key: "token", Value: "drive-token"},
	))

	result := DriveDo(c, "forge", http.MethodPatch, "/pulls/3", `{"state":"closed"}`, "token")
	core.RequireTrue(t, result.OK)
	core.AssertEqual(t, `{"closed":true}`, result.Value.(string))
}
