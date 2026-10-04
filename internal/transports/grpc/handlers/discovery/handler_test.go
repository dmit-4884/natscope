// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	discoverypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/discovery"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

type fakeMicro struct {
	result    *entities.MicroDiscovery
	connID    string
	skipStats bool
}

func (f *fakeMicro) ListServices(_ context.Context, connectionID string, skipStats bool) (*entities.MicroDiscovery, error) {
	f.connID = connectionID
	f.skipStats = skipStats
	return f.result, nil
}

func TestListServices_ConvertsTheDiscovery(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)
	stats := &entities.MicroEndpointStats{
		NumRequests: 4, NumErrors: 1, LastError: "boom",
		ProcessingTime: 40 * time.Millisecond, AverageProcessingTime: 10 * time.Millisecond,
	}
	endpoint := entities.MicroEndpoint{
		Name: "Create", Subject: "orders.create", QueueGroup: "q", Metadata: map[string]string{"k": "v"}, Stats: stats,
		ProtoMethod: &entities.ProtoMethodMatch{SourceID: "src", Service: "shop.OrderService", Method: "Create", InputType: "shop.In", OutputType: "shop.Out"},
	}
	micro := &fakeMicro{result: &entities.MicroDiscovery{
		InfoAccess:  entities.AccessCheck{Status: entities.AccessAllowed, Operation: "publish", Subject: "$SRV.INFO"},
		StatsAccess: &entities.AccessCheck{Status: entities.AccessDenied, Operation: "publish", Subject: "$SRV.STATS"},
		Services: []entities.MicroService{{
			Name: "orders", Description: "Order API", Versions: []string{"1.0.0"},
			Instances: []entities.MicroInstance{
				{ID: "a", Version: "1.0.0", Started: &started, Endpoints: []entities.MicroEndpoint{endpoint}},
				{ID: "b", Version: "1.0.0", Endpoints: []entities.MicroEndpoint{{Name: "Create", Subject: "orders.create"}}},
			},
			Endpoints: []entities.MicroEndpoint{endpoint},
		}},
	}}

	resp, err := New(micro).ListServices(t.Context(), connect.NewRequest(&discoverypb.ListServicesRequest{ConnectionId: "conn-1"}))
	require.NoError(t, err)
	assert.Equal(t, "conn-1", micro.connID)

	msg := resp.Msg
	assert.Equal(t, natspb.AccessStatus_ACCESS_STATUS_ALLOWED, msg.GetInfoAccess().GetStatus())
	assert.Equal(t, natspb.AccessStatus_ACCESS_STATUS_DENIED, msg.GetStatsAccess().GetStatus())
	assert.Equal(t, "$SRV.STATS", msg.GetStatsAccess().GetSubject())

	require.Len(t, msg.GetServices(), 1)
	svc := msg.GetServices()[0]
	assert.Equal(t, "orders", svc.GetName())
	assert.Equal(t, []string{"1.0.0"}, svc.GetVersions())
	require.Len(t, svc.GetInstances(), 2)
	assert.Equal(t, started, svc.GetInstances()[0].GetStarted().AsTime())
	assert.Nil(t, svc.GetInstances()[1].GetStarted(), "no stats → no start time")
	assert.Nil(t, svc.GetInstances()[1].GetEndpoints()[0].Stats, "no stats → unset")

	ep := svc.GetEndpoints()[0]
	assert.Equal(t, "orders.create", ep.GetSubject())
	assert.Equal(t, "q", ep.GetQueueGroup())
	assert.Equal(t, map[string]string{"k": "v"}, ep.GetMetadata())
	assert.Equal(t, int64(4), ep.GetStats().GetNumRequests())
	assert.Equal(t, "boom", ep.GetStats().GetLastError())
	assert.Equal(t, 10*time.Millisecond, ep.GetStats().GetAverageProcessingTime().AsDuration())
	assert.Equal(t, "shop.In", ep.GetProtoMethod().GetInputType())
}

func TestListServices_DeniedInfoLeavesStatsUnset(t *testing.T) {
	t.Parallel()

	micro := &fakeMicro{result: &entities.MicroDiscovery{
		InfoAccess: entities.AccessCheck{Status: entities.AccessDenied, Operation: "subscribe", Subject: "_INBOX.x"},
	}}
	resp, err := New(micro).ListServices(t.Context(), connect.NewRequest(&discoverypb.ListServicesRequest{ConnectionId: "c"}))
	require.NoError(t, err)

	assert.Equal(t, natspb.AccessStatus_ACCESS_STATUS_DENIED, resp.Msg.GetInfoAccess().GetStatus())
	assert.Equal(t, "subscribe", resp.Msg.GetInfoAccess().GetOperation())
	assert.Nil(t, resp.Msg.StatsAccess)
	assert.Empty(t, resp.Msg.GetServices())
}

func TestListServices_PassesSkipStats(t *testing.T) {
	t.Parallel()

	micro := &fakeMicro{result: &entities.MicroDiscovery{InfoAccess: entities.AccessCheck{Status: entities.AccessAllowed}}}
	_, err := New(micro).ListServices(t.Context(), connect.NewRequest(&discoverypb.ListServicesRequest{ConnectionId: "c", SkipStats: true}))
	require.NoError(t, err)

	assert.True(t, micro.skipStats)
}
