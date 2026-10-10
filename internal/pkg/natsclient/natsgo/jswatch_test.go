// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func TestJetStreamWatch_SubjectsFollowTheAPIPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		domain, prefix string
		want           []string
	}{
		{"default", "", "", []string{"$JS.API.STREAM.INFO.ORDERS", "$JS.API.CONSUMER.CREATE.ORDERS."}},
		{"domain", "hub", "", []string{
			"$JS.hub.API.STREAM.INFO.ORDERS", "$JS.API.STREAM.INFO.ORDERS",
			"$JS.hub.API.CONSUMER.CREATE.ORDERS.", "$JS.API.CONSUMER.CREATE.ORDERS.",
		}},
		{"imported prefix with a trailing dot", "", "JS.A.API.", []string{
			"JS.A.API.STREAM.INFO.ORDERS", "$JS.API.STREAM.INFO.ORDERS",
			"JS.A.API.CONSUMER.CREATE.ORDERS.", "$JS.API.CONSUMER.CREATE.ORDERS.",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			w := &jetStreamWatch{api: apiPrefix(tt.domain, tt.prefix)}
			assert.Equal(t, tt.want, w.subjectsFor("ORDERS", subjStreamInfo, subjConsumerCreate))
			assert.Equal(t, w.api+".STREAM.LIST", w.subject(subjStreamList))
			assert.Equal(t, w.api+".CONSUMER.INFO.ORDERS.c", (&Client{api: w.api}).apiSubject("CONSUMER.INFO.ORDERS.c"))
		})
	}
}

// TestJetStreamWatch_ADomainTheServerServesFailsFastWhenDenied checks the calls on the server's own domain, whose API
// the server checks permissions on as the default $JS.API subjects.
func TestJetStreamWatch_ADomainTheServerServesFailsFastWhenDenied(t *testing.T) {
	t.Parallel()
	url := startTestServer(t, func(o *server.Options) {
		o.JetStream, o.JetStreamDomain, o.StoreDir = true, "hub", t.TempDir()
		o.Users = []*server.User{
			{Username: "admin", Password: "pw"},
			{Username: "app", Password: "pw", Permissions: &server.Permissions{Publish: &server.SubjectPermission{
				Allow: []string{">"},
				Deny:  []string{"$JS.API.STREAM.INFO.SECRET", "$JS.API.CONSUMER.LIST.SECRET"},
			}}},
		}
	})
	admin, err := nats.Connect(url, nats.UserInfo("admin", "pw"))
	require.NoError(t, err)
	defer admin.Close()
	js, err := jetstream.New(admin)
	require.NoError(t, err)
	for _, name := range []string{"SECRET", "OPEN"} {
		_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: []string{name + ".>"}})
		require.NoError(t, err)
	}
	_, err = js.CreateConsumer(t.Context(), "SECRET", jetstream.ConsumerConfig{Durable: "reader"})
	require.NoError(t, err)

	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs:       []string{url},
		Auth:       &entities.AuthConfig{Method: entities.AuthMethodUserPass, Username: new("app"), Password: new("pw")},
		Connection: &entities.ConnectionConfig{JetstreamDomain: new("hub")},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	c, ok := conn.(*Client)
	require.True(t, ok)

	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	start := time.Now()
	_, err = c.GetStreamInfo(ctx, "SECRET")
	assert.True(t, errors.Is(err, errs.ErrNATSPermissionViolation), "%v", err)
	assert.Less(t, time.Since(start), 2*time.Second, "stream info waits out its timeout")

	start = time.Now()
	overview, err := c.GetConsumersOverview(ctx)
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 2*time.Second, "the consumers overview waits out its timeout")
	require.Len(t, overview.UnreadableStreams, 1)
	assert.Equal(t, "SECRET", overview.UnreadableStreams[0].Stream)
}
