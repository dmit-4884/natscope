// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/timestamppb"

	messagespb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/messages"
)

type searchRun struct {
	seqs     []uint64
	progress int
	done     *messagespb.SearchDone
}

func search(t *testing.T, env *e2eEnv, req *messagespb.SearchMessagesRequest) searchRun {
	t.Helper()
	stream, err := env.messages.SearchMessages(t.Context(), connect.NewRequest(req))
	require.NoError(t, err)
	defer stream.Close()

	var run searchRun
	for stream.Receive() {
		switch ev := stream.Msg().GetEvent().(type) {
		case *messagespb.SearchMessagesResponse_Progress:
			run.progress++
		case *messagespb.SearchMessagesResponse_Matches:
			for _, m := range ev.Matches.GetMessages() {
				run.seqs = append(run.seqs, m.GetSequence())
			}
		case *messagespb.SearchMessagesResponse_Done:
			run.done = ev.Done
		}
	}
	require.NoError(t, stream.Err())
	require.NotNil(t, run.done, "a run always ends with a summary")
	return run
}

func searchErr(t *testing.T, env *e2eEnv, req *messagespb.SearchMessagesRequest) error {
	t.Helper()
	stream, err := env.messages.SearchMessages(t.Context(), connect.NewRequest(req))
	if err != nil {
		return err
	}
	defer stream.Close()
	for stream.Receive() {
	}
	return stream.Err()
}

func seqRange(from, to uint64) []uint64 {
	var out []uint64
	if from <= to {
		for s := from; s <= to; s++ {
			out = append(out, s)
		}
		return out
	}
	for s := from; s >= to; s-- {
		out = append(out, s)
	}
	return out
}

func seedSearchStream(t *testing.T, js jetstream.JetStream, name string) []time.Time {
	t.Helper()
	addStream(t, js, name, "search."+name+".>")
	stamps := make([]time.Time, 0, 30)
	for i := 1; i <= 30; i++ {
		subject := fmt.Sprintf("search.%s.other", name)
		if i%2 == 1 {
			subject = fmt.Sprintf("search.%s.order", name)
		}
		note := "plain"
		if i == 17 {
			note = "the NeEdLe is here"
		}
		msg := &nats.Msg{Subject: subject, Data: fmt.Appendf(nil, `{"id":"order-%d","note":%q}`, i, note), Header: nats.Header{}}
		if i == 23 {
			msg.Header.Set("X-Trace", "abc")
		}
		ack, err := js.PublishMsg(t.Context(), msg)
		require.NoError(t, err)
		require.Equal(t, uint64(i), ack.Sequence)
		stamps = append(stamps, time.Now())
		time.Sleep(2 * time.Millisecond)
	}
	return stamps
}

func TestSearchMessages(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "search", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	stamps := seedSearchStream(t, js, "S")

	req := func(mut func(r *messagespb.SearchMessagesRequest)) *messagespb.SearchMessagesRequest {
		r := &messagespb.SearchMessagesRequest{ConnectionId: connID, StreamName: "S", Direction: messagespb.Direction_DIRECTION_FORWARD}
		mut(r)
		return r
	}

	t.Run("text is found anywhere in the stream, ignoring case", func(t *testing.T) {
		run := search(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.Text = "needle" }))
		assert.Equal(t, []uint64{17}, run.seqs)
		assert.Equal(t, messagespb.SearchStopReason_SEARCH_STOP_REASON_COMPLETE, run.done.GetReason())
		assert.Equal(t, uint64(30), run.done.GetScanned())
		assert.Equal(t, uint64(1), run.done.GetMatched())
		assert.Equal(t, uint64(1), run.done.GetRangeFirstSeq())
		assert.Equal(t, uint64(30), run.done.GetRangeLastSeq())
		assert.Nil(t, run.done.NextSeq, "a complete run has nothing to continue")
		assert.Positive(t, run.progress, "progress is reported")
	})

	t.Run("a regular expression matches as written", func(t *testing.T) {
		run := search(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.Text = `order-1[0-4]"`; r.Regex = true }))
		assert.Equal(t, seqRange(10, 14), run.seqs)
	})

	t.Run("backward search returns the newest first", func(t *testing.T) {
		run := search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.Text, r.Regex, r.Direction = `order-1[0-4]"`, true, messagespb.Direction_DIRECTION_BACKWARD
		}))
		assert.Equal(t, seqRange(14, 10), run.seqs)
	})

	t.Run("a header is matched by name without regard to case, then by value", func(t *testing.T) {
		assert.Equal(t, []uint64{23}, search(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.HeaderName = "x-trace" })).seqs)
		assert.Equal(t, []uint64{23}, search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.HeaderName, r.HeaderValue = "X-Trace", "abc"
		})).seqs)
		assert.Empty(t, search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.HeaderName, r.HeaderValue = "X-Trace", "other"
		})).seqs)
	})

	t.Run("the subject filter is applied by the server", func(t *testing.T) {
		run := search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.SubjectFilter = new("search.S.other")
			r.Text = "order-1"
		}))
		assert.Equal(t, []uint64{10, 12, 14, 16, 18}, run.seqs)
		assert.Equal(t, uint64(15), run.done.GetScanned(), "only the filtered subject is read")
	})

	t.Run("sequence and time windows bound the search", func(t *testing.T) {
		bySeq := search(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.FromSeq, r.ToSeq = new(uint64(5)), new(uint64(9)) }))
		assert.Equal(t, seqRange(5, 9), bySeq.seqs)
		assert.Equal(t, uint64(5), bySeq.done.GetScanned())

		byTime := search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.FromTime = timestamppb.New(stamps[19])
			r.ToTime = timestamppb.New(stamps[24])
		}))
		assert.Equal(t, seqRange(21, 25), byTime.seqs)
	})

	t.Run("a malformed regular expression is rejected", func(t *testing.T) {
		err := searchErr(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.Text, r.Regex = "order-(", true }))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("an inverted range is rejected", func(t *testing.T) {
		err := searchErr(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.FromSeq, r.ToSeq = new(uint64(9)), new(uint64(5)) }))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("direct reads find the same messages", func(t *testing.T) {
		setMessagesFetchMethod(t, env, "direct")
		assert.Equal(t, []uint64{17}, search(t, env, req(func(r *messagespb.SearchMessagesRequest) { r.Text = "needle" })).seqs)
		assert.Equal(t, []uint64{10, 12, 14, 16, 18}, search(t, env, req(func(r *messagespb.SearchMessagesRequest) {
			r.SubjectFilter = new("search.S.other")
			r.Text = "order-1"
		})).seqs)
	})
}

func TestSearchMessagesContinues(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "search-more", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	addStream(t, js, "MANY", "many.>")
	for i := 1; i <= 600; i++ {
		js.PublishAsync("many.hit", fmt.Appendf(nil, `{"hit":%d}`, i))
	}
	select {
	case <-js.PublishAsyncComplete():
	case <-time.After(10 * time.Second):
		t.Fatal("publishing did not finish")
	}

	for _, tt := range []struct {
		name      string
		direction messagespb.Direction
		first     []uint64
		next      uint64
		rest      []uint64
	}{
		{name: "forward", direction: messagespb.Direction_DIRECTION_FORWARD, first: seqRange(1, 500), next: 501, rest: seqRange(501, 600)},
		{name: "backward", direction: messagespb.Direction_DIRECTION_BACKWARD, first: seqRange(600, 101), next: 100, rest: seqRange(100, 1)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			base := &messagespb.SearchMessagesRequest{ConnectionId: connID, StreamName: "MANY", Direction: tt.direction, Text: "hit"}
			first := search(t, env, base)
			assert.Equal(t, tt.first, first.seqs)
			assert.Equal(t, messagespb.SearchStopReason_SEARCH_STOP_REASON_MATCH_LIMIT, first.done.GetReason())
			require.NotNil(t, first.done.NextSeq)
			assert.Equal(t, tt.next, first.done.GetNextSeq())

			more := &messagespb.SearchMessagesRequest{
				ConnectionId: connID, StreamName: "MANY", Direction: tt.direction, Text: "hit", CursorSeq: first.done.NextSeq,
			}
			rest := search(t, env, more)
			assert.Equal(t, tt.rest, rest.seqs)
			assert.Equal(t, messagespb.SearchStopReason_SEARCH_STOP_REASON_COMPLETE, rest.done.GetReason())
			assert.Equal(t, uint64(1), rest.done.GetRangeFirstSeq(), "the range stays the whole search, not the rest")
			assert.Equal(t, uint64(600), rest.done.GetRangeLastSeq())
		})
	}
}

func TestSearchMessagesWorkQueue(t *testing.T) {
	env := setupE2E(t)
	connID := createTestConnection(t, env, "search-wq", env.natsURL, nil)
	js := jetStreamFor(t, env.natsURL)
	_, err := js.CreateStream(t.Context(), jetstream.StreamConfig{Name: "JOBS", Subjects: []string{"jobs.>"}, Retention: jetstream.WorkQueuePolicy})
	require.NoError(t, err)
	publishAll(t, js, "jobs.a", "jobs.b", "jobs.a")

	run := search(t, env, &messagespb.SearchMessagesRequest{ConnectionId: connID, StreamName: "JOBS", Text: "jobs.a"})
	assert.ElementsMatch(t, []uint64{1, 3}, run.seqs, "a work queue is searched without a consumer that would drain it")

	info, err := js.Stream(t.Context(), "JOBS")
	require.NoError(t, err)
	assert.Equal(t, uint64(3), info.CachedInfo().State.Msgs, "searching left the queue untouched")
}
