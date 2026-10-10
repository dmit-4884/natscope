// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"strconv"
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestConsumersOverview_OneListCallPerStream(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	defer nc.Close()
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	const streams = 3
	for i := range streams {
		name := "S" + strconv.Itoa(i)
		_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: []string{"s" + strconv.Itoa(i) + ".>"}})
		require.NoError(t, err)
		_, err = js.CreateOrUpdateConsumer(t.Context(), name, jetstream.ConsumerConfig{Durable: "worker"})
		require.NoError(t, err)
	}

	c := dialClient(t, url)
	before, err := js.AccountInfo(t.Context())
	require.NoError(t, err)
	overview, err := c.GetConsumersOverview(t.Context())
	require.NoError(t, err)
	after, err := js.AccountInfo(t.Context())
	require.NoError(t, err)

	assert.Len(t, overview.Consumers, streams)
	assert.Len(t, overview.Streams, streams)
	assert.LessOrEqual(t, after.API.Total-before.API.Total, uint64(1+streams+1), "the stream list, one consumer list per stream, and this account info")
}

func TestConsumersOverview_ConcurrentCallsShareOneListing(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	js := rawJetStream(t, url)
	const streams = 20
	for i := range streams {
		name := "S" + strconv.Itoa(i)
		_, err := js.CreateStream(t.Context(), jetstream.StreamConfig{Name: name, Subjects: []string{"s" + strconv.Itoa(i) + ".>"}})
		require.NoError(t, err)
		_, err = js.CreateOrUpdateConsumer(t.Context(), name, jetstream.ConsumerConfig{Durable: "worker"})
		require.NoError(t, err)
	}
	c := dialClient(t, url)
	before, err := js.AccountInfo(t.Context())
	require.NoError(t, err)

	const callers = 20
	start := make(chan struct{})
	var wg sync.WaitGroup
	overviews := make([]*entities.ConsumersOverview, callers)
	errors := make([]error, callers)
	for i := range callers {
		wg.Go(func() {
			<-start
			overviews[i], errors[i] = c.GetConsumersOverview(t.Context())
		})
	}
	close(start)
	wg.Wait()
	after, err := js.AccountInfo(t.Context())
	require.NoError(t, err)

	for i := range callers {
		require.NoError(t, errors[i])
		assert.Len(t, overviews[i].Consumers, streams)
	}
	perListing := uint64(1 + streams)
	assert.Less(t, after.API.Total-before.API.Total, perListing*callers/2, "concurrent callers share a listing in flight")
}
