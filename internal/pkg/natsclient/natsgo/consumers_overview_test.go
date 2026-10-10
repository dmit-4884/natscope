// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"sync/atomic"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConsumersOverview_ListsConsumersOnlyOfStreamsThatHaveThem(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	for _, name := range []string{"A", "B", "C"} {
		_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: []string{name}})
		require.NoError(t, err)
	}
	_, err = js.CreateConsumer(t.Context(), "B", jetstream.ConsumerConfig{Durable: "worker"})
	require.NoError(t, err)
	var lists atomic.Int32
	sub, err := nc.Subscribe("$JS.API.CONSUMER.LIST.>", func(*nats.Msg) { lists.Add(1) })
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Unsubscribe() })
	require.NoError(t, nc.Flush())

	overview, err := dialClient(t, url).GetConsumersOverview(t.Context())

	require.NoError(t, err)
	require.Len(t, overview.Consumers, 1)
	assert.Equal(t, "worker", overview.Consumers[0].Name)
	assert.Len(t, overview.Streams, 3)
	require.NoError(t, nc.Flush())
	assert.Equal(t, int32(1), lists.Load())
}
