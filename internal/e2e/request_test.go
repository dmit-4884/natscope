// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/durationpb"

	publishpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/publish"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// TestRequestReply drives RequestMessage against a live responder, a subject nobody listens on,
// and a responder that never answers.
func TestRequestReply(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "request-reply", env.natsURL, nil)

	nc, err := nats.Connect(env.natsURL)
	require.NoError(t, err)
	defer nc.Close()

	_, err = nc.Subscribe("svc.echo", func(m *nats.Msg) {
		reply := nats.NewMsg(m.Reply)
		reply.Data = append([]byte("echo: "), m.Data...)
		reply.Header.Set("Echo-Trace", m.Header.Get("Trace"))
		_ = m.RespondMsg(reply)
	})
	require.NoError(t, err)
	_, err = nc.Subscribe("svc.silent", func(*nats.Msg) {})
	require.NoError(t, err)
	require.NoError(t, nc.Flush())

	t.Run("reply", func(t *testing.T) {
		resp, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.echo",
			Data:         "ping",
			Headers:      map[string]string{"Trace": "t-1"},
		}))
		require.NoError(t, err)
		assert.Equal(t, []byte("echo: ping"), resp.Msg.GetData())
		assert.Equal(t, "t-1", resp.Msg.GetHeaders()["Echo-Trace"])
		assert.True(t, strings.HasPrefix(resp.Msg.GetSubject(), nats.InboxPrefix), "reply subject %q", resp.Msg.GetSubject())
		assert.Positive(t, resp.Msg.GetDuration().AsDuration())
	})

	t.Run("empty payload", func(t *testing.T) {
		resp, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.echo",
		}))
		require.NoError(t, err)
		assert.Equal(t, []byte("echo: "), resp.Msg.GetData())
	})

	t.Run("no responders", func(t *testing.T) {
		start := time.Now()
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.nobody",
			Timeout:      durationpb.New(10 * time.Second),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "NATS_NO_RESPONDERS", errorReason(t, err))
		assert.Less(t, time.Since(start), 5*time.Second, "no responders must fail fast, not wait for the timeout")
	})

	t.Run("timeout", func(t *testing.T) {
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.silent",
			Timeout:      durationpb.New(200 * time.Millisecond),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeDeadlineExceeded, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "NATS_TIMEOUT", errorReason(t, err))
	})

	t.Run("wildcard subject", func(t *testing.T) {
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.*",
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "NATS_INVALID_ARGUMENT", errorReason(t, err))
	})

	t.Run("timeout above the limit", func(t *testing.T) {
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.echo",
			Timeout:      durationpb.New(2 * time.Minute),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", err)
	})

	t.Run("publish permission denied", func(t *testing.T) {
		srv := startNATSWithOptions(t, func(o *server.Options) {
			o.Users = []*server.User{{
				Username:    "req",
				Password:    "pw",
				Permissions: &server.Permissions{Publish: &server.SubjectPermission{Deny: []string{"secret.>"}}},
			}}
		})
		restricted := createTestConnection(t, env, "request-restricted", srv.ClientURL(), &natstypes.AuthConfig{
			Method:   natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD,
			Username: new("req"),
			Password: new("pw"),
		})

		start := time.Now()
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: restricted,
			Subject:      "secret.op",
			Timeout:      durationpb.New(10 * time.Second),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err), "%v", err)
		assert.Equal(t, "NATS_PERMISSION_VIOLATION", errorReason(t, err))
		assert.Less(t, time.Since(start), 5*time.Second, "a denied publish must fail fast, not wait for the timeout")
	})

	t.Run("proto type without source", func(t *testing.T) {
		_, err := env.publish.RequestMessage(ctx, connect.NewRequest(&publishpb.RequestMessageRequest{
			ConnectionId: connID,
			Subject:      "svc.echo",
			Data:         "{}",
			MessageType:  new("pkg.Ping"),
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "%v", err)
	})
}
