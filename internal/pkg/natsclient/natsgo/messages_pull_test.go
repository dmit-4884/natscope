// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

type refusingConsumer struct {
	jetstream.Consumer
	refusals int
	pullErr  error
	msgs     []jetstream.Msg
	pulls    int
}

func (c *refusingConsumer) Fetch(int, ...jetstream.FetchOpt) (jetstream.MessageBatch, error) {
	c.pulls++
	if c.pulls <= c.refusals {
		return closedBatch(nil, nats.ErrNoResponders), nil
	}
	return closedBatch(c.msgs, c.pullErr), nil
}

type fakeBatch struct {
	msgs chan jetstream.Msg
	err  error
}

func (b *fakeBatch) Messages() <-chan jetstream.Msg { return b.msgs }

func (b *fakeBatch) Error() error { return b.err }

func closedBatch(msgs []jetstream.Msg, err error) *fakeBatch {
	ch := make(chan jetstream.Msg, len(msgs))
	for _, m := range msgs {
		ch <- m
	}
	close(ch)
	return &fakeBatch{msgs: ch, err: err}
}

type subjectMsg struct {
	jetstream.Msg
	subject string
}

func (m subjectMsg) Subject() string { return m.subject }

type seqMsg struct {
	jetstream.Msg
	seq uint64
}

func (m seqMsg) Subject() string { return "s" }

func (m seqMsg) Data() []byte { return nil }

func (m seqMsg) Headers() nats.Header { return nil }

func (m seqMsg) Metadata() (*jetstream.MsgMetadata, error) {
	return &jetstream.MsgMetadata{Sequence: jetstream.SequencePair{Consumer: 1, Stream: m.seq}}, nil
}

type consumerStream struct {
	jetstream.Stream
	consumer *refusingConsumer
}

func (s consumerStream) CreateConsumer(context.Context, jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	return s.consumer, nil
}

func (s consumerStream) DeleteConsumer(context.Context, string) error { return nil }

func (c *refusingConsumer) CachedInfo() *jetstream.ConsumerInfo {
	return &jetstream.ConsumerInfo{NumPending: 1}
}

func TestFetchBrowseWindow_ReadsPastARefusedPull(t *testing.T) {
	t.Parallel()
	consumer := &refusingConsumer{refusals: 1, msgs: []jetstream.Msg{seqMsg{seq: 7}}}

	got, err := (&Client{}).fetchBrowseWindow(t.Context(), consumer, 1, time.Second, func(uint64) bool { return false })

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, uint64(7), got[0].Sequence)
}

func TestScanConsumerFrom_ReadsPastARefusedPull(t *testing.T) {
	t.Parallel()
	stream := consumerStream{consumer: &refusingConsumer{refusals: 1, msgs: []jetstream.Msg{seqMsg{seq: 7}}}}
	var seqs []uint64
	var scanErr error

	(&Client{}).scanConsumerFrom(t.Context(), stream, entities.ScanOptions{FromSeq: 7, ToSeq: 7}, 7, time.Second,
		func(msg *entities.Message, err error) bool {
			if err != nil {
				scanErr = err
				return false
			}
			seqs = append(seqs, msg.Sequence)
			return true
		})

	require.NoError(t, scanErr)
	assert.Equal(t, []uint64{7}, seqs)
}

type refusingStream struct {
	jetstream.Stream
}

func (refusingStream) GetMsg(context.Context, uint64, ...jetstream.GetMsgOpt) (*jetstream.RawStreamMsg, error) {
	return nil, nats.ErrNoResponders
}

func TestNoAnswer_IsATimeoutNotADisabledJetStream(t *testing.T) {
	t.Parallel()
	c := &Client{}
	always := &refusingConsumer{refusals: 1 << 30}

	_, browseErr := c.fetchBrowseWindow(t.Context(), always, 1, time.Second, func(uint64) bool { return false })
	var scanErr error
	c.scanConsumerFrom(t.Context(), consumerStream{consumer: always}, entities.ScanOptions{FromSeq: 1, ToSeq: 1}, 1, time.Second,
		func(_ *entities.Message, err error) bool {
			scanErr = err
			return false
		})
	_, getErr := c.getMsgWithRetry(t.Context(), refusingStream{}, 1)

	for name, err := range map[string]error{"browse": browseErr, "scan": scanErr, "get": getErr} {
		require.ErrorIs(t, err, errs.ErrNATSTimeout, name)
		assert.NotErrorIs(t, err, errs.ErrJetStreamNotEnabled, name)
	}
}

func pulledSubjects(batch pulledBatch) []string {
	var got []string
	for msg := range batch.All() {
		got = append(got, msg.Subject())
	}
	return got
}

func TestFetchPull_PullsAgainWhileNoResponderAnswers(t *testing.T) {
	t.Parallel()
	consumer := &refusingConsumer{refusals: 2, msgs: []jetstream.Msg{subjectMsg{subject: "a"}, subjectMsg{subject: "b"}}}

	batch, err := fetchPull(t.Context(), consumer, 10, time.Second)

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, pulledSubjects(batch))
	require.NoError(t, batch.Error())
	assert.Equal(t, 3, consumer.pulls)
}

func TestFetchPull_KeepsAnEmptyWaitAsOnePull(t *testing.T) {
	t.Parallel()
	consumer := &refusingConsumer{pullErr: nats.ErrTimeout}

	batch, err := fetchPull(t.Context(), consumer, 10, time.Second)

	require.NoError(t, err)
	assert.Empty(t, pulledSubjects(batch))
	require.ErrorIs(t, batch.Error(), nats.ErrTimeout)
	assert.Equal(t, 1, consumer.pulls)
}
