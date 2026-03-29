// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_RegisterHTTPTransport_Good(t *testing.T) {
	c := core.New()

	RegisterHTTPTransport(c)

	assert.Contains(t, c.API().Protocols(), "http")
	assert.Contains(t, c.API().Protocols(), "https")
}

func TestTransport_HTTPGet_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	result := HTTPGet(context.Background(), srv.URL, "test-token", "token")

	require.True(t, result.OK)
	assert.Equal(t, `{"status":"ok"}`, result.Value.(string))
}

func TestTransport_DriveGet_Good(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/core/go-io", r.URL.Path)
		assert.Equal(t, "Bearer drive-token", r.Header.Get("Authorization"))
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

	require.True(t, result.OK)
	assert.Equal(t, `{"repo":"go-io"}`, result.Value.(string))
}

func TestTransport_DriveDo_Bad_MissingDrive(t *testing.T) {
	result := DriveDo(core.New(), "missing", "PATCH", "/repos/core/go-io", `{"title":"AX"}`, "token")

	assert.False(t, result.OK)
	assert.Error(t, result.Value.(error))
}

func TestTransport_HTTPDo_Ugly_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"upstream"}`))
	}))
	defer srv.Close()

	result := HTTPDelete(context.Background(), srv.URL, "", "", "")

	assert.False(t, result.OK)
	assert.Equal(t, `{"error":"upstream"}`, result.Value.(string))
}

func TestTransport_RegisterHTTPTransport_Ugly_StreamRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "token api-token", r.Header.Get("Authorization"))
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
	require.True(t, streamResult.OK)

	stream := streamResult.Value.(core.Stream)
	require.NoError(t, stream.Send([]byte(`{"ping":1}`)))

	response, err := stream.Receive()
	require.NoError(t, err)
	assert.Equal(t, `{"echo":"ok"}`, string(response))
	assert.NoError(t, stream.Close())
}
