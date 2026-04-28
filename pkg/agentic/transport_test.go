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
