// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package stats

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	statspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/stats"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
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

	t.Run("Capabilities", func(t *testing.T) {
		t.Parallel()

		svc := &serverInfoNATSSvc{
			getServerInfoResult: &entities.ServerInfo{
				Version:   "2.14.2",
				Jetstream: true,
				Capabilities: &entities.ServerCapabilities{
					ApiLevel: 4, ConsumerPause: true, MessageTtl: true, AtomicPublish: true,
					PriorityGroups: true, MsgCounters: true, MsgSchedules: true, PriorityPrioritized: true,
					AsyncPersist: true, ConsumerReset: true, CronSchedules: true, BatchPublish: true,
				},
			},
		}
		h := New(svc, svc)

		resp, err := h.GetServerInfo(t.Context(), connect.NewRequest(&statspb.GetServerInfoRequest{
			ConnectionId: "conn1",
		}))

		require.NoError(t, err)
		caps := resp.Msg.ServerInfo.GetCapabilities()
		require.NotNil(t, caps)
		assert.Equal(t, int32(4), caps.GetApiLevel())
		assert.True(t, caps.GetConsumerPause())
		assert.True(t, caps.GetMessageTtl())
		assert.True(t, caps.GetAtomicPublish())
		assert.True(t, caps.GetPriorityGroups())
		assert.True(t, caps.GetMsgCounters())
		assert.True(t, caps.GetMsgSchedules())
		assert.True(t, caps.GetPriorityPrioritized())
		assert.True(t, caps.GetAsyncPersist())
		assert.True(t, caps.GetConsumerReset())
		assert.True(t, caps.GetCronSchedules())
		assert.True(t, caps.GetBatchPublish())
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

type overviewNATSSvc struct {
	natssvc.StatsReader
	natssvc.ConnectionManager
	overview *entities.ConsumersOverview
}

func (m *overviewNATSSvc) GetConsumersOverview(_ context.Context, _ string) (*entities.ConsumersOverview, error) {
	return m.overview, nil
}

func TestHandler_GetAllConsumers(t *testing.T) {
	t.Parallel()
	active := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	svc := &overviewNATSSvc{overview: &entities.ConsumersOverview{
		Consumers: []entities.ConsumerInfo{{
			Name: "worker", Stream: "ORDERS", NumPending: 3,
			Delivered: entities.SequenceInfo{Consumer: 2, Stream: 7, LastActive: &active},
			Config:    &entities.ConsumerConfig{FilterSubject: "orders.created", MaxAckPending: 10},
		}},
		Streams: []entities.StreamInfo{{Config: entities.StreamConfig{Name: "ORDERS", MaxMsgs: 100}, State: &entities.StreamState{Msgs: 95}}},
		UnreadableStreams: []entities.UnreadableStream{
			{Stream: "SECRET", Access: &entities.AccessCheck{Status: entities.AccessDenied, Operation: "publish", Subject: "$JS.API.CONSUMER.LIST.SECRET"}},
			{Stream: "BROKEN", Error: "stream offline"},
		},
	}}

	resp, err := New(svc, svc).GetAllConsumers(t.Context(), connect.NewRequest(&statspb.GetAllConsumersRequest{ConnectionId: "conn1"}))
	require.NoError(t, err)

	require.Len(t, resp.Msg.GetConsumers(), 1)
	c := resp.Msg.GetConsumers()[0]
	assert.Equal(t, "worker", c.GetName())
	assert.Equal(t, "ORDERS", c.GetStream())
	assert.Equal(t, uint64(3), c.GetNumPending())
	assert.Equal(t, uint64(7), c.GetDelivered().GetStream())
	assert.Equal(t, active, c.GetDelivered().GetLastActive().AsTime())
	assert.Nil(t, c.GetAckFloor().GetLastActive())
	assert.Equal(t, int32(10), c.GetConfig().GetMaxAckPending())

	require.Len(t, resp.Msg.GetStreams(), 1)
	assert.Equal(t, "ORDERS", resp.Msg.GetStreams()[0].GetConfig().GetName())
	assert.Equal(t, uint64(95), resp.Msg.GetStreams()[0].GetState().GetMsgs())

	require.Len(t, resp.Msg.GetUnreadableStreams(), 2)
	denied := resp.Msg.GetUnreadableStreams()[0]
	assert.Equal(t, "SECRET", denied.GetStream())
	assert.Equal(t, natspb.AccessStatus_ACCESS_STATUS_DENIED, denied.GetAccess().GetStatus())
	assert.Equal(t, "$JS.API.CONSUMER.LIST.SECRET", denied.GetAccess().GetSubject())
	broken := resp.Msg.GetUnreadableStreams()[1]
	assert.Nil(t, broken.Access)
	assert.Equal(t, "stream offline", broken.GetError())
}
