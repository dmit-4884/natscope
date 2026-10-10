// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/connections"
	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
	streamspb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/streams"
	natstypes "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

const hubConf = `
listen: "127.0.0.1:-1"
jetstream { store_dir: %q, domain: hub }
leafnodes { listen: %q }
accounts {
  A: {
    jetstream: enabled
    users: [ { user: owner, password: owner } ]
    exports: [ { service: "$JS.API.>" } ]
  }
  B: {
    users: [ { user: guest, password: guest } ]
    imports: [ { service: { account: A, subject: "$JS.API.>" }, to: "JS.A.API.>" } ]
  }
}
`

const leafConf = `
listen: "127.0.0.1:-1"
jetstream { store_dir: %q, domain: leaf }
leafnodes { remotes: [ { url: "nats-leaf://owner:owner@%s" } ] }
`

func startConfNATS(t *testing.T, conf string) *server.Server {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nats.conf")
	require.NoError(t, os.WriteFile(path, []byte(conf), 0o600))
	opts, err := server.ProcessConfigFile(path)
	require.NoError(t, err)
	opts.NoLog, opts.NoSigs = true, true
	srv, err := server.NewServer(opts)
	require.NoError(t, err)
	go srv.Start()
	require.True(t, srv.ReadyForConnections(10*time.Second))
	t.Cleanup(srv.Shutdown)
	return srv
}

// startHubAndLeaf starts a hub in JetStream domain "hub" and a leaf node in domain "leaf" connected to it.
func startHubAndLeaf(t *testing.T) (hubURL, leafURL string) {
	t.Helper()
	leafAddr := freeLocalAddr(t)
	hub := startConfNATS(t, fmt.Sprintf(hubConf, t.TempDir(), leafAddr))
	leaf := startConfNATS(t, fmt.Sprintf(leafConf, t.TempDir(), leafAddr))
	require.Eventually(t, func() bool { return leaf.NumLeafNodes() == 1 }, 10*time.Second, 50*time.Millisecond)
	return hub.ClientURL(), leaf.ClientURL()
}

func jsConnection(t *testing.T, env *e2eEnv, name, url, user string, cfg *natstypes.ConnectionConfig) string {
	t.Helper()
	req := &connectionspb.CreateConnectionRequest{Name: name, Urls: []string{url}, Connection: cfg}
	if user != "" {
		req.Auth = &natstypes.AuthConfig{Method: natstypes.AuthMethod_AUTH_METHOD_USER_PASSWORD, Username: &user, Password: &user}
	}
	resp, err := env.connections.CreateConnection(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	return resp.Msg.GetConnection().GetId()
}

// TestJetStreamDomainAndAPIPrefix manages JetStream through a domain and through an imported API prefix.
func TestJetStreamDomainAndAPIPrefix(t *testing.T) {
	env := setupE2E(t)
	ctx := t.Context()
	hubURL, leafURL := startHubAndLeaf(t)

	nc, err := nats.Connect(hubURL, nats.UserInfo("owner", "owner"))
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{Name: "ORDERS", Subjects: []string{"orders.>"}})
	require.NoError(t, err)
	_, err = js.Publish(ctx, "orders.1", []byte("first"))
	require.NoError(t, err)

	readsOrders := func(t *testing.T, connID string) {
		t.Helper()
		list, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: connID}))
		require.NoError(t, err)
		assert.True(t, containsStream(list.Msg.GetStreams(), "ORDERS"))
		msg, err := env.messages.GetMessage(ctx, connect.NewRequest(&messagespb.GetMessageRequest{
			ConnectionId: connID, StreamName: "ORDERS", Sequence: 1,
		}))
		require.NoError(t, err)
		assert.Equal(t, "orders.1", msg.Msg.GetMessage().GetSubject())
	}

	t.Run("a leaf node reaches the hub's JetStream through its domain", func(t *testing.T) {
		plain := jsConnection(t, env, "leaf-local", leafURL, "", nil)
		list, err := env.streams.ListStreams(ctx, connect.NewRequest(&streamspb.ListStreamsRequest{ConnectionId: plain}))
		require.NoError(t, err)
		assert.False(t, containsStream(list.Msg.GetStreams(), "ORDERS"), "the leaf's own JetStream has no ORDERS")

		domain := jsConnection(t, env, "leaf-hub", leafURL, "", &natstypes.ConnectionConfig{JetstreamDomain: new("hub")})
		readsOrders(t, domain)

		_, err = env.management.CreateConsumer(ctx, connect.NewRequest(&managementpb.CreateConsumerRequest{
			ConnectionId: domain, StreamName: "ORDERS", Name: "audit",
		}))
		require.NoError(t, err)
		consumers, err := env.management.ListConsumers(ctx, connect.NewRequest(&managementpb.ListConsumersRequest{
			ConnectionId: domain, StreamName: "ORDERS",
		}))
		require.NoError(t, err)
		require.Len(t, consumers.Msg.GetConsumers(), 1)
		_, err = env.management.PurgeStream(ctx, connect.NewRequest(&managementpb.PurgeStreamRequest{
			ConnectionId: domain, StreamName: "ORDERS", Keep: new(uint64(1)),
		}))
		require.NoError(t, err)
	})

	t.Run("another account reaches JetStream through an imported API prefix", func(t *testing.T) {
		prefixed := jsConnection(t, env, "guest-prefix", hubURL, "guest", &natstypes.ConnectionConfig{JetstreamApiPrefix: new("JS.A.API")})
		readsOrders(t, prefixed)
	})

	t.Run("a domain and a prefix together are refused", func(t *testing.T) {
		_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
			Name: "both", Urls: []string{hubURL},
			Connection: &natstypes.ConnectionConfig{JetstreamDomain: new("hub"), JetstreamApiPrefix: new("JS.A.API")},
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("domains and prefixes follow the server's rules", func(t *testing.T) {
		create := func(name string, cfg *natstypes.ConnectionConfig) error {
			_, err := env.connections.CreateConnection(ctx, connect.NewRequest(&connectionspb.CreateConnectionRequest{
				Name: name, Urls: []string{hubURL}, Connection: cfg,
			}))
			return err
		}
		require.NoError(t, create("colon-domain", &natstypes.ConnectionConfig{JetstreamDomain: new("hub:east")}))
		require.NoError(t, create("accented-domain", &natstypes.ConnectionConfig{JetstreamDomain: new("région")}))
		for name, cfg := range map[string]*natstypes.ConnectionConfig{
			"dotted-domain": {JetstreamDomain: new("hub.east")},
			"empty-token":   {JetstreamApiPrefix: new("JS..API")},
			"leading-dot":   {JetstreamApiPrefix: new(".JS.API")},
		} {
			err := create(name, cfg)
			require.Error(t, err, name)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), name)
		}
	})
}
