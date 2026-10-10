// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func workQueueWithJobs(t *testing.T, jobs int) (*Client, jetstream.Stream) {
	t.Helper()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	stream, err := js.CreateStream(t.Context(), jetstream.StreamConfig{
		Name: "JOBS", Subjects: []string{"jobs.>"}, Retention: jetstream.WorkQueuePolicy,
	})
	require.NoError(t, err)
	for i := range jobs {
		subject := "jobs.a"
		if i%2 == 1 {
			subject = "jobs.b"
		}
		_, err = js.Publish(t.Context(), subject, nil)
		require.NoError(t, err)
	}
	return dialClient(t, url), stream
}

func TestGetMessages_ReadsAWorkQueueWithoutConsumingIt(t *testing.T) {
	t.Parallel()
	c, stream := workQueueWithJobs(t, 4)
	now := time.Now().Add(-time.Hour)

	cases := map[string]entities.GetMessagesOptions{
		"page":     {Limit: 10, FetchMethod: fetchMethodConsumer},
		"filtered": {Limit: 10, FetchMethod: fetchMethodConsumer, SubjectFilter: "jobs.b"},
		"jump":     {Limit: 10, FetchMethod: fetchMethodConsumer, StartTime: &now, Direction: "forward"},
	}
	want := map[string][]uint64{"page": {4, 3, 2, 1}, "filtered": {4, 2}, "jump": {1, 2, 3, 4}}

	for name, opts := range cases {
		resp, err := c.GetMessages(t.Context(), "JOBS", opts)
		require.NoError(t, err, name)
		assert.Equal(t, want[name], browseSeqs(resp), name)
	}
	info, err := stream.Info(t.Context())
	require.NoError(t, err)
	assert.Equal(t, uint64(4), info.State.Msgs)
}
