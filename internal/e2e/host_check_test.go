// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHostHeaderAllowlist covers QA-003: the transport used to accept any
// Host header on a loopback bind, so a page whose hostname resolves to
// 127.0.0.1 via DNS rebinding could send same-origin requests (matching
// Sec-Fetch-Site/Origin, so http.CrossOriginProtection alone lets them
// through) that the real client would never send. A request whose Host
// doesn't match 127.0.0.1/[::1]/localhost at the bound port must be rejected
// before it reaches any handler.
func TestHostHeaderAllowlist(t *testing.T) {
	env := setupE2E(t)

	req, err := http.NewRequest(http.MethodPost, //nolint:noctx // deliberately no context; this is a raw transport-layer probe
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

// TestHostHeaderAllowlist_AcceptsRealHost is the control: a request whose
// Host matches the bound address must still work normally.
func TestHostHeaderAllowlist_AcceptsRealHost(t *testing.T) {
	env := setupE2E(t)

	req, err := http.NewRequest(http.MethodPost, //nolint:noctx // deliberately no context; this is a raw transport-layer probe
		env.baseURL+"/natscope.nats.connections.v1.ConnectionsService/ListConnections",
		bytes.NewBufferString("{}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck

	require.Equal(t, http.StatusOK, resp.StatusCode)
}
