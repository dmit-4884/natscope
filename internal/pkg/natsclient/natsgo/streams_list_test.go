// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"testing"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStreams_LeavesTheServerJSONToSingleStreamReads(t *testing.T) {
	t.Parallel()
	_, url := jetStreamServer(t)
	_, err := rawJetStream(t, url).CreateStream(t.Context(), jetstream.StreamConfig{Name: "LIST", Subjects: []string{"list.>"}})
	require.NoError(t, err)
	c := dialClient(t, url)

	streams, err := c.ListStreams(t.Context())
	require.NoError(t, err)
	stream, err := c.GetStreamInfo(t.Context(), "LIST")
	require.NoError(t, err)

	require.Len(t, streams, 1)
	assert.Equal(t, "LIST", streams[0].Config.Name)
	assert.Empty(t, streams[0].Raw, "a list of hundreds of streams must not carry each stream's JSON")
	assert.Contains(t, stream.Raw, `"name":"LIST"`)
}
