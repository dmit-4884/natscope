// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package mcpcmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointFor(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"127.0.0.1:4280": "http://127.0.0.1:4280/mcp",
		"0.0.0.0:9090":   "http://127.0.0.1:9090/mcp",
		"[::]:9090":      "http://127.0.0.1:9090/mcp",
		":8080":          "http://127.0.0.1:8080/mcp",
		"[::1]:4280":     "http://[::1]:4280/mcp",
		"localhost:4280": "http://localhost:4280/mcp",
		"":               "http://127.0.0.1:4280/mcp",
	}
	for addr, want := range tests {
		assert.Equal(t, want, endpointFor(addr), addr)
	}
}

func TestBasicAuthTransport(t *testing.T) {
	t.Parallel()
	var user, pass string
	var ok bool
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		user, pass, ok = r.BasicAuth()
	}))
	defer srv.Close()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL, nil)
	require.NoError(t, err)
	resp, err := (&http.Client{Transport: basicAuth{user: "admin", pass: "pw", next: http.DefaultTransport}}).Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	assert.True(t, ok)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pw", pass)
	_, _, reqHasAuth := req.BasicAuth()
	assert.False(t, reqHasAuth, "the caller's request must not be mutated")
}
