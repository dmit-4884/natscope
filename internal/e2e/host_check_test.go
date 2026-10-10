// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHostHeaderAllowlist checks that a foreign Host header is rejected before any handler runs.
func TestHostHeaderAllowlist(t *testing.T) {
	env := setupE2E(t)

	req, err := http.NewRequest(http.MethodPost, //nolint:noctx // raw transport probe
		env.baseURL+"/natscope.nats.connections.v1.ConnectionsService/ListConnections",
		bytes.NewBufferString("{}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Host = "evil.example:1"
	req.Header.Set("Origin", "http://evil.example:1")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusMisdirectedRequest, resp.StatusCode,
		"a request with an untrusted Host header must be rejected before reaching any handler")
}

// TestHostHeaderAllowlist_AcceptsRealHost checks that the bound IP is accepted as Host.
func TestHostHeaderAllowlist_AcceptsRealHost(t *testing.T) {
	env := setupE2E(t)

	req, err := http.NewRequest(http.MethodPost, //nolint:noctx // raw transport probe
		env.baseURL+"/natscope.nats.connections.v1.ConnectionsService/ListConnections",
		bytes.NewBufferString("{}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
