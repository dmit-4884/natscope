// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/rpc/errdetails"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func TestNATSErrors_UnknownConnection(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	cases := []struct {
		name string
		call func() error
	}{
		{"streams.ListStreams", func() error {
			_, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: unknownUUID}))
			return err
		}},
		{"management.ListKVBuckets", func() error {
			_, err := env.management.ListKVBuckets(ctx, connect.NewRequest(&managementpb.ListKVBucketsRequest{ConnectionId: unknownUUID}))
			return err
		}},
		{"management.CreateStream", func() error {
			_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
				ConnectionId: unknownUUID, Name: "S1",
			}))
			return err
		}},
		{"stats.GetHealth", func() error {
			_, err := env.stats.GetHealth(ctx, connect.NewRequest(&statspb.GetHealthRequest{ConnectionId: unknownUUID}))
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "%v", err)
			assert.Equal(t, "CONNECTION_NOT_FOUND", errorReason(t, err))
		})
	}
}

func TestNATSErrors_JetStreamDisabled(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	srv := startNATSWithOptions(t, func(*server.Options) {})
	connID := createTestConnection(t, env, "no-jetstream", srv.ClientURL(), nil)

	cases := []struct {
		name string
		call func() error
	}{
		{"streams.ListStreams", func() error {
			_, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: connID}))
			return err
		}},
		{"management.ListKVBuckets", func() error {
			_, err := env.management.ListKVBuckets(ctx, connect.NewRequest(&managementpb.ListKVBucketsRequest{ConnectionId: connID}))
			return err
		}},
		{"management.ListObjectBuckets", func() error {
			_, err := env.management.ListObjectBuckets(ctx, connect.NewRequest(&managementpb.ListObjectBucketsRequest{ConnectionId: connID}))
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "%v", err)
			assert.Equal(t, "NATS_JETSTREAM_NOT_ENABLED", errorReason(t, err))
		})
	}
}

func TestNATSErrors_AuthorizationViolation(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	srv := startNATSWithOptions(t, func(o *server.Options) {
		o.Username = "qa"
		o.Password = "right-password"
	})
	connID := createTestConnection(t, env, "bad-password", srv.ClientURL(), &natstypes.AuthConfig{
		Method:   natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD,
		Username: new("qa"),
		Password: new("wrong-password"),
	})

	_, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: connID}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err), "%v", err)
	assert.Equal(t, "NATS_AUTHORIZATION_VIOLATION", errorReason(t, err))
}

func TestNATSErrors_ServerErrorClasses(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	connID := createTestConnection(t, env, "error-classes", env.natsURL, nil)

	_, err := env.management.CreateKVBucket(ctx, connect.NewRequest(&managementpb.CreateKVBucketRequest{
		ConnectionId: connID,
		Config:       &natstypes.KVBucketConfig{Bucket: "tiny", MaxBytes: 1},
	}))
	require.NoError(t, err)

	cases := []struct {
		name     string
		wantCode connect.Code
		call     func() error
	}{
		{"invalid stream config", connect.CodeInvalidArgument, func() error {
			_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
				ConnectionId: connID, Name: "ROLLUP", Storage: 1, AllowRollup: true, DenyPurge: true,
			}))
			return err
		}},
		{"replicas without cluster", connect.CodeFailedPrecondition, func() error {
			_, err := env.management.CreateStream(ctx, connect.NewRequest(&managementpb.CreateStreamRequest{
				ConnectionId: connID, Name: "R3", Replicas: 3,
			}))
			return err
		}},
		{"value above server max_payload", connect.CodeInvalidArgument, func() error {
			_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: "tiny", Key: "big", Value: strings.Repeat("!", 1<<20+1),
			}))
			return err
		}},
		{"bucket max_bytes exceeded", connect.CodeResourceExhausted, func() error {
			_, err := env.management.PutKVKey(ctx, connect.NewRequest(&managementpb.PutKVKeyRequest{
				ConnectionId: connID, Bucket: "tiny", Key: "k", Value: "value",
			}))
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			require.Error(t, err)
			assert.Equal(t, tc.wantCode, connect.CodeOf(err), "%v", err)
		})
	}
}

func TestNATSErrors_HealthWhileReconnecting(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()

	srv := startNATSWithOptions(t, func(*server.Options) {})
	connID := createTestConnection(t, env, "reconnecting", srv.ClientURL(), nil)

	resp, err := env.stats.GetHealth(ctx, connect.NewRequest(&statspb.GetHealthRequest{ConnectionId: connID}))
	require.NoError(t, err)
	require.Equal(t, "connected", resp.Msg.GetHealth().GetStatus())

	srv.Shutdown()

	require.Eventually(t, func() bool {
		resp, err := env.stats.GetHealth(ctx, connect.NewRequest(&statspb.GetHealthRequest{ConnectionId: connID}))
		return err == nil && resp.Msg.GetHealth().GetIsReconnecting() && resp.Msg.GetHealth().GetStatus() == "reconnecting"
	}, 10*time.Second, 100*time.Millisecond)
}

func startNATSWithOptions(t *testing.T, configure func(*server.Options)) *server.Server {
	t.Helper()

	opts := &server.Options{
		Host:   "127.0.0.1",
		Port:   -1,
		NoLog:  true,
		NoSigs: true,
	}
	configure(opts)

	srv, err := server.NewServer(opts)
	require.NoError(t, err, "create embedded NATS server")

	go srv.Start()

	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("embedded NATS server not ready within 10s")
	}
	t.Cleanup(srv.Shutdown)
	return srv
}

func createTestConnection(t *testing.T, env *e2eEnv, name, url string, auth *natstypes.AuthConfig) string {
	t.Helper()

	resp, err := env.connections.CreateConnection(t.Context(), connect.NewRequest(&connectionspb.CreateConnectionRequest{
		Name: name,
		Urls: []string{url},
		Auth: auth,
	}))
	require.NoError(t, err)
	return resp.Msg.GetConnection().GetId()
}

func errorReason(t *testing.T, err error) string {
	t.Helper()

	connectErr, ok := errors.AsType[*connect.Error](err)
	require.True(t, ok, "expected a connect error, got %v", err)
	for _, d := range connectErr.Details() {
		v, valErr := d.Value()
		if valErr != nil {
			continue
		}
		if info, ok := v.(*errdetails.ErrorInfo); ok {
			return info.GetReason()
		}
	}
	return ""
}
