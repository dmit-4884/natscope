// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	discoverypb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/discovery"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

func startOrdersService(t *testing.T, nc *nats.Conn, version string) {
	t.Helper()
	svc, err := micro.AddService(nc, micro.Config{Name: "orders", Version: version, Description: "Order API"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = svc.Stop() })
	require.NoError(t, svc.AddEndpoint("Create", micro.HandlerFunc(func(r micro.Request) {
		_ = r.Respond([]byte(`{"ok":true}`))
	}), micro.WithEndpointSubject("orders.create")))
}

func listServices(t *testing.T, env *e2eEnv, connID string) *discoverypb.ListServicesResponse {
	t.Helper()
	resp, err := env.discovery.ListServices(t.Context(), connect.NewRequest(&discoverypb.ListServicesRequest{ConnectionId: connID}))
	require.NoError(t, err)
	return resp.Msg
}

func restrictedConnection(t *testing.T, env *e2eEnv, name string, perms *server.Permissions) string {
	t.Helper()
	srv := startNATSWithOptions(t, func(o *server.Options) {
		o.Users = []*server.User{{Username: "u", Password: "pw", Permissions: perms}}
	})
	return createTestConnection(t, env, name, srv.ClientURL(), &natstypes.AuthConfig{
		Method:   natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD,
		Username: new("u"),
		Password: new("pw"),
	})
}

func TestDiscovery(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "discovery", env.natsURL, nil)

	t.Run("no running service is an allowed empty list, answered at once", func(t *testing.T) {
		start := time.Now()
		got := listServices(t, env, connID)
		assert.Less(t, time.Since(start), 1500*time.Millisecond, "no responders must not wait for the timeout")
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_ALLOWED, got.GetInfoAccess().GetStatus())
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_ALLOWED, got.GetStatsAccess().GetStatus())
		assert.Empty(t, got.GetServices())
	})

	t.Run("groups the instances of a running service", func(t *testing.T) {
		nc1, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		defer nc1.Close()
		nc2, err := nats.Connect(env.natsURL)
		require.NoError(t, err)
		defer nc2.Close()
		startOrdersService(t, nc1, "1.0.0")
		startOrdersService(t, nc2, "1.1.0")

		_, err = nc1.Request("orders.create", []byte(`{}`), 2*time.Second)
		require.NoError(t, err)

		start := time.Now()
		got := listServices(t, env, connID)
		assert.Less(t, time.Since(start), 1500*time.Millisecond, "discovery ends shortly after the replies stop")
		require.Len(t, got.GetServices(), 1)
		svc := got.GetServices()[0]
		assert.Equal(t, "orders", svc.GetName())
		assert.Equal(t, "Order API", svc.GetDescription())
		assert.Equal(t, []string{"1.0.0", "1.1.0"}, svc.GetVersions())
		assert.Len(t, svc.GetInstances(), 2)
		require.Len(t, svc.GetEndpoints(), 1)
		ep := svc.GetEndpoints()[0]
		assert.Equal(t, "orders.create", ep.GetSubject())
		assert.Equal(t, int64(1), ep.GetStats().GetNumRequests(), "stats are summed across instances")
	})
}

func TestDiscoveryAccess(t *testing.T) {
	env := setupE2E(t)

	t.Run("no permission to ask services is no access, not an empty list", func(t *testing.T) {
		connID := restrictedConnection(t, env, "discovery-denied", &server.Permissions{
			Publish: &server.SubjectPermission{Deny: []string{"$SRV.>"}},
		})
		start := time.Now()
		got := listServices(t, env, connID)
		assert.Less(t, time.Since(start), 1500*time.Millisecond, "a denied publish must fail fast")
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_DENIED, got.GetInfoAccess().GetStatus())
		assert.Equal(t, "publish", got.GetInfoAccess().GetOperation())
		assert.Equal(t, "$SRV.INFO", got.GetInfoAccess().GetSubject())
		assert.Nil(t, got.StatsAccess)
	})

	t.Run("a denied reply inbox is no access too", func(t *testing.T) {
		connID := restrictedConnection(t, env, "discovery-no-inbox", &server.Permissions{
			Subscribe: &server.SubjectPermission{Deny: []string{"_INBOX.>"}},
		})
		got := listServices(t, env, connID)
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_DENIED, got.GetInfoAccess().GetStatus())
		assert.Equal(t, "subscribe", got.GetInfoAccess().GetOperation())
	})

	t.Run("statistics alone can be denied", func(t *testing.T) {
		connID := restrictedConnection(t, env, "discovery-no-stats", &server.Permissions{
			Publish: &server.SubjectPermission{Deny: []string{"$SRV.STATS", "$SRV.STATS.>"}},
		})
		got := listServices(t, env, connID)
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_ALLOWED, got.GetInfoAccess().GetStatus())
		assert.Equal(t, natstypes.AccessStatus_ACCESS_STATUS_DENIED, got.GetStatsAccess().GetStatus())
		assert.Equal(t, "$SRV.STATS", got.GetStatsAccess().GetSubject())
	})
}
