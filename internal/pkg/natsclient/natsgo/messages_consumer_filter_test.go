// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"slices"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
)

func streamWithRareSubject(t *testing.T, total uint64, rareSeqs ...uint64) *Client {
	t.Helper()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	_, err = js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "S", Subjects: []string{"busy", "rare"}})
	require.NoError(t, err)
	for seq := uint64(1); seq <= total; seq++ {
		subject := "busy"
		if slices.Contains(rareSeqs, seq) {
			subject = "rare"
		}
		_, err = js.PublishAsync(subject, nil)
		require.NoError(t, err)
	}
	<-js.PublishAsyncComplete()
	return dialClient(t, url)
}

func browseSeqs(resp *entities.MessagesResponse) []uint64 {
	seqs := make([]uint64, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		seqs = append(seqs, msg.Sequence)
	}
	return seqs
}

func TestGetMessagesViaConsumer_FindsOldMatchesOfARareSubject(t *testing.T) {
	t.Parallel()
	c := streamWithRareSubject(t, 12000, 3, 10)

	resp, err := c.GetMessages(t.Context(), "S", entities.GetMessagesOptions{SubjectFilter: "rare", Limit: 50, FetchMethod: fetchMethodConsumer})

	require.NoError(t, err)
	assert.Equal(t, []uint64{10, 3}, browseSeqs(resp))
	assert.False(t, resp.HasMore)
}

func TestGetMessagesViaConsumer_NarrowsAnOldBurstOfMatches(t *testing.T) {
	t.Parallel()
	burst := make([]uint64, 0, 3000)
	for seq := uint64(1); seq <= 3000; seq++ {
		burst = append(burst, seq)
	}
	c := streamWithRareSubject(t, 12000, burst...)

	resp, err := c.GetMessages(t.Context(), "S", entities.GetMessagesOptions{SubjectFilter: "rare", Limit: 2, FetchMethod: fetchMethodConsumer})

	require.NoError(t, err)
	assert.Equal(t, []uint64{3000, 2999}, browseSeqs(resp))
	assert.True(t, resp.HasMore)
	assert.Equal(t, uint64(2998), resp.NextSeq)
}

func TestGetMessagesViaConsumer_PagesBackThroughSparseMatches(t *testing.T) {
	t.Parallel()
	c := streamWithRareSubject(t, 12000, 3, 10, 6000, 11990)
	opts := entities.GetMessagesOptions{SubjectFilter: "rare", Limit: 2, FetchMethod: fetchMethodConsumer}

	first, err := c.GetMessages(t.Context(), "S", opts)
	require.NoError(t, err)
	assert.Equal(t, []uint64{11990, 6000}, browseSeqs(first))
	require.True(t, first.HasMore)

	opts.StartSeq = first.NextSeq
	second, err := c.GetMessages(t.Context(), "S", opts)
	require.NoError(t, err)
	assert.Equal(t, []uint64{10, 3}, browseSeqs(second))
	assert.False(t, second.HasMore)
}
