// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func streamWithConsumer(t *testing.T, url string, messages int) jetstream.Stream {
	t.Helper()
	js := rawJetStream(t, url)
	s, err := js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "ACT", Subjects: []string{"act.>"}})
	require.NoError(t, err)
	for range messages {
		_, err = js.Publish(t.Context(), "act.x", []byte("m"))
		require.NoError(t, err)
	}
	_, err = s.CreateConsumer(t.Context(), jetstream.ConsumerConfig{Durable: "worker", AckPolicy: jetstream.AckExplicitPolicy})
	require.NoError(t, err)
	return s
}

func TestPauseConsumer_RefusesATimeInThePast(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	s := streamWithConsumer(t, url, 0)
	c := dialClient(t, url)

	_, err := c.PauseConsumer(t.Context(), "ACT", "worker", time.Now().Add(-time.Hour).Format(time.RFC3339))

	require.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
	assert.Contains(t, err.Error(), "future")
	info, err := s.Consumer(t.Context(), "worker")
	require.NoError(t, err)
	assert.False(t, info.CachedInfo().Paused)
}

func TestResetConsumer_RefusesASequenceBeyondTheStream(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	streamWithConsumer(t, url, 3)
	c := dialClient(t, url)

	_, err := c.ResetConsumer(t.Context(), "ACT", "worker", new(uint64(100)))

	require.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
	assert.Contains(t, err.Error(), "last sequence 3")
}

func TestResetConsumer_AcceptsTheNextSequence(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	streamWithConsumer(t, url, 3)
	c := dialClient(t, url)

	resp, err := c.ResetConsumer(t.Context(), "ACT", "worker", new(uint64(4)))

	require.NoError(t, err)
	assert.Equal(t, uint64(4), resp.ResetSeq)
}

func TestPurgeStream_RefusesSequenceWithKeep(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	streamWithConsumer(t, url, 3)
	c := dialClient(t, url)

	_, err := c.PurgeStream(t.Context(), "ACT", entities.StreamPurgeRequest{Sequence: 2, Keep: 1})

	require.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
	assert.Contains(t, err.Error(), "keep")
}
