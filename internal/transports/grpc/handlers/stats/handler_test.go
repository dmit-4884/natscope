// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
)

// --- Mocks ---

// serverInfoNATSSvc is a minimal mock for GetServerInfo testing.
type serverInfoNATSSvc struct {
	natssvc.StatsReader
	natssvc.ConnectionManager
	getServerInfoResult *entities.ServerInfo
	getServerInfoErr    error
}

func (m *serverInfoNATSSvc) GetServerInfo(_ context.Context, _ string) (*entities.ServerInfo, error) {
	return m.getServerInfoResult, m.getServerInfoErr
}

// --- Tests ---

func TestHandler_GetServerInfo(t *testing.T) {
	t.Parallel()

	t.Run("Success_WithJetStream", func(t *testing.T) {
		t.Parallel()

		svc := &serverInfoNATSSvc{
			getServerInfoResult: &entities.ServerInfo{
				ServerId:     "server-1",
				ServerName:   "nats-1",
				Version:      "2.12.0",
				Host:         "127.0.0.1",
				Port:         4222,
				ClusterName:  "my-cluster",
				MaxPayload:   1048576,
				ConnectedUrl: "nats://127.0.0.1:4222",
				Jetstream:    true,
				JsAccount: &entities.JetStreamAccountInfo{
					Memory:        1024000,
					Storage:       5000000,
					Streams:       5,
					Consumers:     10,
					MemoryLimit:   -1,
					StorageLimit:  -1,
					StreamLimit:   -1,
					ConsumerLimit: -1,
				},
			},
		}
		h := New(svc, svc)

		resp, err := h.GetServerInfo(t.Context(), connect.NewRequest(&statspb.GetServerInfoRequest{
			ConnectionId: "conn1",
		}))

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.NotNil(t, resp.Msg.ServerInfo)

		si := resp.Msg.ServerInfo
		assert.Equal(t, "server-1", si.ServerId)
		assert.Equal(t, "nats-1", si.ServerName)
		assert.Equal(t, "2.12.0", si.Version)
		assert.Equal(t, "127.0.0.1", si.Host)
		assert.Equal(t, int32(4222), si.Port)
		assert.Equal(t, "my-cluster", si.ClusterName)
		assert.Equal(t, int64(1048576), si.MaxPayload)
		assert.Equal(t, "nats://127.0.0.1:4222", si.ConnectedUrl)
		assert.True(t, si.Jetstream)

		require.NotNil(t, si.JsAccount)
		assert.Equal(t, int64(1024000), si.JsAccount.Memory)
		assert.Equal(t, int64(5000000), si.JsAccount.Storage)
		assert.Equal(t, int64(5), si.JsAccount.Streams)
		assert.Equal(t, int64(10), si.JsAccount.Consumers)
	})

	t.Run("WithoutJetStream", func(t *testing.T) {
		t.Parallel()

		svc := &serverInfoNATSSvc{
			getServerInfoResult: &entities.ServerInfo{
				ServerId:  "server-2",
				Version:   "2.12.0",
				Jetstream: false,
			},
		}
		h := New(svc, svc)

		resp, err := h.GetServerInfo(t.Context(), connect.NewRequest(&statspb.GetServerInfoRequest{
			ConnectionId: "conn1",
		}))

		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.False(t, resp.Msg.ServerInfo.Jetstream)
		assert.Nil(t, resp.Msg.ServerInfo.JsAccount)
	})

	t.Run("ServiceError", func(t *testing.T) {
		t.Parallel()

		svc := &serverInfoNATSSvc{
			getServerInfoErr: assert.AnError,
		}
		h := New(svc, svc)

		resp, err := h.GetServerInfo(t.Context(), connect.NewRequest(&statspb.GetServerInfoRequest{
			ConnectionId: "conn1",
		}))

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}
