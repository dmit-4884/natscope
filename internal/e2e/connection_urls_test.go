// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
)

// TestConnectionURLForms accepts the URL forms nats.go dials and rejects foreign schemes.
func TestConnectionURLForms(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	accepted := []string{
		"localhost:4222",
		"nats.example.com:6745",
		"user:secret@nats.example.com:4222",
		"[::1]:4222",
		"nats://nats.example.com:4222",
		"wss://nats.example.com/ws",
	}
	for i, u := range accepted {
		_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "url-ok-" + string(rune('a'+i)), Urls: []string{u},
		}))
		assert.NoError(t, err, "%q must be accepted", u)
	}

	rejected := []string{"http://nats.example.com:4222", "file:///etc/passwd", "javascript:alert(1)", "host:port"}
	for i, u := range rejected {
		_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "url-bad-" + string(rune('a'+i)), Urls: []string{u},
		}))
		require.Error(t, err, "%q must be rejected", u)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%q: %v", u, err)
	}

	resp, err := env.connections.TestConnection(ctx, connect.NewRequest(&connectionspb.TestConnectionRequest{
		Urls: []string{strings.TrimPrefix(env.natsURL, "nats://")},
	}))
	require.NoError(t, err)
	assert.True(t, resp.Msg.GetSuccess(), "schemeless test must connect: %s", resp.Msg.GetError())
}
