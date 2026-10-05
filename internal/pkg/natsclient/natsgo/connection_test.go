// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestSubscribe_HidesTheConnectionsOwnInboxUnlessNamed(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs:       []string{url},
		Connection: &entities.ConnectionConfig{InboxPrefix: ptr.Wrap("_INBOX_alice")},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	var mu sync.Mutex
	got := map[string][]string{}
	collect := func(pattern string) {
		_, subErr := conn.Subscribe(t.Context(), pattern, func(msg *entities.NatsMessage) {
			mu.Lock()
			got[pattern] = append(got[pattern], msg.Subject)
			mu.Unlock()
		}, nil)
		require.NoError(t, subErr)
	}
	collect(">")
	collect("_INBOX_alice.>")

	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	require.NoError(t, conn.(*Client).conn.Flush())
	require.NoError(t, nc.Publish("_INBOX_alice.abc.1", nil))
	require.NoError(t, nc.Publish("orders.created", nil))
	require.NoError(t, nc.Flush())

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got[">"]) > 0 && len(got["_INBOX_alice.>"]) > 0
	}, 5*time.Second, 10*time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, []string{"orders.created"}, got[">"])
	assert.Equal(t, []string{"_INBOX_alice.abc.1"}, got["_INBOX_alice.>"])
}

func TestSubscribe_SaysWhichSubjectsItDelivers(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	conn, err := NewDialer().Dial(t.Context(), &entities.SavedConnection{
		URLs:       []string{url},
		Connection: &entities.ConnectionConfig{InboxPrefix: ptr.Wrap("_INBOX_alice")},
	})
	require.NoError(t, err)
	t.Cleanup(conn.Close)

	all, err := conn.Subscribe(t.Context(), ">", func(*entities.NatsMessage) {}, nil)
	require.NoError(t, err)
	inbox, err := conn.Subscribe(t.Context(), "_INBOX_alice.>", func(*entities.NatsMessage) {}, nil)
	require.NoError(t, err)

	assert.True(t, all.Delivers("orders.created"))
	assert.False(t, all.Delivers("_INBOX_alice.abc.1"))
	assert.True(t, inbox.Delivers("_INBOX_alice.abc.1"))
	assert.False(t, inbox.Delivers("orders.created"))
}
