// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package micro

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

type fakeDiscoverer struct {
	natssvc.ServiceDiscoverer
	infos, stats      []entities.MicroReport
	infoErr, statsErr error
	statsCalls        int
}

func (f *fakeDiscoverer) MicroInfo(context.Context, string) ([]entities.MicroReport, error) {
	return f.infos, f.infoErr
}

func (f *fakeDiscoverer) MicroStats(context.Context, string) ([]entities.MicroReport, error) {
	f.statsCalls++
	return f.stats, f.statsErr
}

type fakeRegistry struct {
	protosvc.Registry
	types    []entities.SchemaType
	services map[string]*entities.SchemaService
}

func (f *fakeRegistry) ListTypes(context.Context, string) ([]entities.SchemaType, error) {
	return f.types, nil
}

func (f *fakeRegistry) DescribeType(_ context.Context, _, _, fullName string, _ bool) (*entities.TypeDescription, error) {
	svc, ok := f.services[fullName]
	if !ok {
		return nil, errs.ErrProtoTypeNotFound
	}
	return &entities.TypeDescription{Services: []*entities.SchemaService{svc}}, nil
}

func denied(subject string) error {
	return &errs.NATSPermissionError{Operation: errs.PermissionOperationPublish, Subject: subject}
}

var started = time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)

func TestListServices_GroupsInstancesAndSumsStats(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{
		infos: []entities.MicroReport{
			{Name: "orders", ID: "b", Version: "1.1.0", Description: "Order API", Endpoints: []entities.MicroEndpoint{
				{Name: "Create", Subject: "orders.create", QueueGroup: "q"},
			}},
			{Name: "orders", ID: "a", Version: "1.0.0", Description: "Order API", Endpoints: []entities.MicroEndpoint{
				{Name: "Create", Subject: "orders.create", QueueGroup: "q"},
			}},
			{Name: "billing", ID: "c", Version: "2.0.0"},
		},
		stats: []entities.MicroReport{
			{Name: "orders", ID: "a", Started: started, Endpoints: []entities.MicroEndpoint{{Name: "Create", Subject: "orders.create",
				Stats: &entities.MicroEndpointStats{NumRequests: 3, NumErrors: 1, LastError: "boom", ProcessingTime: 30 * time.Millisecond}}}},
			{Name: "orders", ID: "b", Started: started, Endpoints: []entities.MicroEndpoint{{Name: "Create", Subject: "orders.create",
				Stats: &entities.MicroEndpointStats{NumRequests: 1, ProcessingTime: 10 * time.Millisecond}}}},
		},
	}
	svc := New(nats, &fakeRegistry{})

	got, err := svc.ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	assert.Equal(t, entities.AccessCheck{Status: entities.AccessAllowed, Operation: "publish", Subject: "$SRV.INFO"}, got.InfoAccess)
	require.NotNil(t, got.StatsAccess)
	assert.Equal(t, entities.AccessAllowed, got.StatsAccess.Status)

	require.Len(t, got.Services, 2)
	assert.Equal(t, "billing", got.Services[0].Name, "services sorted by name")
	orders := got.Services[1]
	assert.Equal(t, "Order API", orders.Description)
	assert.Equal(t, []string{"1.0.0", "1.1.0"}, orders.Versions)
	require.Len(t, orders.Instances, 2)
	assert.Equal(t, "a", orders.Instances[0].ID, "instances sorted by ID")
	require.NotNil(t, orders.Instances[0].Started)
	assert.Equal(t, started, *orders.Instances[0].Started)
	require.Len(t, orders.Endpoints, 1)
	assert.Equal(t, &entities.MicroEndpointStats{
		NumRequests:           4,
		NumErrors:             1,
		LastError:             "boom",
		ProcessingTime:        40 * time.Millisecond,
		AverageProcessingTime: 10 * time.Millisecond,
	}, orders.Endpoints[0].Stats)
	assert.Nil(t, got.Services[0].Instances[0].Started, "an instance missing from STATS has no stats")
}

func TestListServices_NoInfoAccessSkipsStats(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{infoErr: denied("$SRV.INFO")}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	assert.Equal(t, entities.AccessCheck{Status: entities.AccessDenied, Operation: "publish", Subject: "$SRV.INFO"}, got.InfoAccess)
	assert.Nil(t, got.StatsAccess)
	assert.Empty(t, got.Services)
	assert.Zero(t, nats.statsCalls, "no doomed STATS request after INFO was denied")
}

func TestListServices_DeniedReplyInboxReadsAsNoAccess(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{infoErr: &errs.NATSPermissionError{Operation: errs.PermissionOperationSubscribe, Subject: "_INBOX.x.y"}}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	assert.Equal(t, entities.AccessCheck{Status: entities.AccessDenied, Operation: "subscribe", Subject: "_INBOX.x.y"}, got.InfoAccess)
}

func TestListServices_StatsDeniedKeepsServicesWithoutStats(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{
		infos:    []entities.MicroReport{{Name: "orders", ID: "a", Endpoints: []entities.MicroEndpoint{{Name: "Create", Subject: "orders.create"}}}},
		statsErr: denied("$SRV.STATS"),
	}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	assert.Equal(t, entities.AccessAllowed, got.InfoAccess.Status)
	require.NotNil(t, got.StatsAccess)
	assert.Equal(t, entities.AccessCheck{Status: entities.AccessDenied, Operation: "publish", Subject: "$SRV.STATS"}, *got.StatsAccess)
	require.Len(t, got.Services, 1)
	assert.Nil(t, got.Services[0].Endpoints[0].Stats)
}

func TestListServices_SkipStatsAsksOnlyForInfo(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{
		infos: []entities.MicroReport{{Name: "orders", ID: "a", Endpoints: []entities.MicroEndpoint{{Name: "Create", Subject: "orders.create"}}}},
		stats: []entities.MicroReport{{Name: "orders", ID: "a", Started: started}},
	}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", true)
	require.NoError(t, err)

	assert.Zero(t, nats.statsCalls, "a refresh after STATS was denied does not ask again")
	assert.Nil(t, got.StatsAccess)
	require.Len(t, got.Services, 1)
	assert.Nil(t, got.Services[0].Instances[0].Started)
}

func TestListServices_OtherFailuresAreErrors(t *testing.T) {
	t.Parallel()

	_, err := New(&fakeDiscoverer{infoErr: errs.ErrNATSConnectionClosed}, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.ErrorIs(t, err, errs.ErrNATSConnectionClosed)

}

func TestListServices_UnreadableStatsKeepTheServices(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{infos: []entities.MicroReport{{Name: "orders", ID: "a"}}, statsErr: errors.New("boom")}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	require.NotNil(t, got.StatsAccess)
	assert.Equal(t, entities.AccessUnspecified, got.StatsAccess.Status)
	assert.Equal(t, "$SRV.STATS", got.StatsAccess.Subject)
	require.Len(t, got.Services, 1)
}

func TestListServices_SortsVersionsBySemver(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{infos: []entities.MicroReport{
		{Name: "orders", ID: "a", Version: "1.10.0"},
		{Name: "orders", ID: "b", Version: "dev"},
		{Name: "orders", ID: "c", Version: "1.9.0"},
		{Name: "orders", ID: "d", Version: "1.2.0"},
	}}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", true)
	require.NoError(t, err)

	require.Len(t, got.Services, 1)
	assert.Equal(t, []string{"1.2.0", "1.9.0", "1.10.0", "dev"}, got.Services[0].Versions)
}

func TestListServices_InstancesCarryRoundTripAndRawReplies(t *testing.T) {
	t.Parallel()

	nats := &fakeDiscoverer{
		infos: []entities.MicroReport{{Name: "orders", ID: "a", RTT: 3 * time.Millisecond, Raw: `{"info":1}`}},
		stats: []entities.MicroReport{{Name: "orders", ID: "a", Started: started, Raw: `{"stats":1}`}},
	}
	got, err := New(nats, &fakeRegistry{}).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	instance := got.Services[0].Instances[0]
	assert.Equal(t, 3*time.Millisecond, instance.RTT)
	assert.JSONEq(t, `{"info":1}`, instance.InfoJSON)
	assert.JSONEq(t, `{"stats":1}`, instance.StatsJSON)
}

func TestListServices_MatchesEndpointsToProtoMethods(t *testing.T) {
	t.Parallel()

	registry := &fakeRegistry{
		types: []entities.SchemaType{
			{FullName: "shop.v1.OrderService", Kind: entities.SchemaTypeService, SourceID: "src-1"},
			{FullName: "shop.v1.AdminService", Kind: entities.SchemaTypeService, SourceID: "src-1"},
			{FullName: "shop.v1.Order", Kind: entities.SchemaTypeMessage, SourceID: "src-1"},
			{FullName: "google.Dep", Kind: entities.SchemaTypeService, SourceID: "src-1", Dependency: true},
		},
		services: map[string]*entities.SchemaService{
			"shop.v1.OrderService": {FullName: "shop.v1.OrderService", Methods: []*entities.SchemaMethod{
				{Name: "CreateOrder", InputType: "shop.v1.CreateOrderRequest", OutputType: "shop.v1.Order"},
				{Name: "Ping", InputType: "shop.v1.PingRequest", OutputType: "shop.v1.Pong"},
			}},
			"shop.v1.AdminService": {FullName: "shop.v1.AdminService", Methods: []*entities.SchemaMethod{
				{Name: "Ping", InputType: "shop.v1.AdminPing", OutputType: "shop.v1.Pong"},
			}},
		},
	}
	nats := &fakeDiscoverer{infos: []entities.MicroReport{{Name: "orders", ID: "a", Endpoints: []entities.MicroEndpoint{
		{Name: "createorder", Subject: "orders.create"},
		{Name: "Ping", Subject: "orders.ping"},
		{Name: "Unknown", Subject: "orders.unknown"},
	}}}}

	got, err := New(nats, registry).ListServices(t.Context(), "conn", false)
	require.NoError(t, err)

	endpoints := got.Services[0].Endpoints
	require.Len(t, endpoints, 3)
	assert.Equal(t, &entities.ProtoMethodMatch{
		SourceID: "src-1", Service: "shop.v1.OrderService", Method: "CreateOrder",
		InputType: "shop.v1.CreateOrderRequest", OutputType: "shop.v1.Order",
	}, endpoints[0].ProtoMethod, "method names match case-insensitively")
	require.NotNil(t, endpoints[1].ProtoMethod, "an ambiguous method name is settled by the service name")
	assert.Equal(t, "shop.v1.OrderService", endpoints[1].ProtoMethod.Service)
	assert.Nil(t, endpoints[2].ProtoMethod)
	assert.Equal(t, endpoints[0].ProtoMethod, got.Services[0].Instances[0].Endpoints[0].ProtoMethod)
}
