// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var timeBase = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// minute returns timeBase + n minutes, so message time increases with sequence (matching monotonic timestamps).
func minute(n int) time.Time { return timeBase.Add(time.Duration(n) * time.Minute) }

// fakeNextPresent builds a nextPresentFunc over fixed present sequences (time = base + seq min), returning the first present seq at/after the requested one.
func fakeNextPresent(present ...uint64) nextPresentFunc {
	sorted := append([]uint64(nil), present...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return func(seq uint64) (uint64, time.Time, bool, error) {
		for _, s := range sorted {
			if s >= seq {
				return s, timeBase.Add(time.Duration(s) * time.Minute), true, nil
			}
		}
		return 0, time.Time{}, false, nil
	}
}

func TestSearchSeqByTime_Contiguous(t *testing.T) {
	t.Parallel()
	// Sequences 1..10, time(seq) = base + seq min.
	at := fakeNextPresent(1, 2, 3, 4, 5, 6, 7, 8, 9, 10)

	tests := []struct {
		name   string
		target time.Time
		want   uint64
	}{
		{"exact hit middle", minute(5), 5},
		{"exact hit first", minute(1), 1},
		{"exact hit last", minute(10), 10},
		{"before first", minute(1).Add(-time.Second), 1},
		{"after last → past end", minute(10).Add(time.Second), 11},
		{"between 5 and 6 → 6", minute(5).Add(30 * time.Second), 6},
		{"between 1 and 2 → 2", minute(1).Add(time.Second), 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := searchSeqByTime(1, 10, tt.target, at)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSearchSeqByTime_WithHoles(t *testing.T) {
	t.Parallel()
	// Interior deletes: 3,4,7,8 are gaps. Present: 1,2,5,6,9,10.
	at := fakeNextPresent(1, 2, 5, 6, 9, 10)

	tests := []struct {
		name   string
		target time.Time
		want   uint64
	}{
		{"target in a gap (minute 4) → next present 5", minute(4), 5},
		{"target equals present 5", minute(5), 5},
		{"target in gap (minute 7) → next present 9", minute(7), 9},
		{"target in gap (minute 8) → next present 9", minute(8), 9},
		{"before first present", minute(1).Add(-time.Second), 1},
		{"exact first present", minute(1), 1},
		{"after last present → past end", minute(10).Add(time.Second), 11},
		{"between 2 and 5, just after 2 → 5", minute(2).Add(time.Second), 5},
		{"exact last present", minute(10), 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := searchSeqByTime(1, 10, tt.target, at)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSearchSeqByTime_WideGaps proves the next-by-subject probe jumps large
// interior gaps without per-sequence scanning (only 2 present msgs in 1M seqs).
func TestSearchSeqByTime_WideGaps(t *testing.T) {
	t.Parallel()
	// Present only at 1 and 1_000_000; everything in between is a gap (deletes /
	// per-subject purge / MaxMsgs pruning). time(1)=min1, time(1e6)=min1e6.
	at := fakeNextPresent(1, 1_000_000)

	tests := []struct {
		name   string
		target time.Time
		want   uint64
	}{
		{"before everything → first present", minute(0), 1},
		{"target inside the giant gap → jumps to far end", minute(500_000), 1_000_000},
		{"target == far end time", minute(1_000_000), 1_000_000},
		{"after far end → past end", minute(1_000_000).Add(time.Second), 1_000_001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Bound probe count: a linear scan would be ~1e6 calls; log2(1e6)≈20.
			calls := 0
			counting := func(seq uint64) (uint64, time.Time, bool, error) {
				calls++
				return at(seq)
			}
			got, err := searchSeqByTime(1, 1_000_000, tt.target, counting)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, calls, 40, "probe count must be O(log N), not O(N)")
		})
	}
}

func TestSearchSeqByTime_SingleMessage(t *testing.T) {
	t.Parallel()
	at := fakeNextPresent(7)

	before, err := searchSeqByTime(7, 7, minute(7).Add(-time.Second), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), before, "target before the only message → that message")

	exact, err := searchSeqByTime(7, 7, minute(7), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), exact)

	after, err := searchSeqByTime(7, 7, minute(7).Add(time.Second), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(8), after, "target after the only message → past end")
}

func TestSearchSeqByTime_EmptyRange(t *testing.T) {
	t.Parallel()
	// lastSeq < firstSeq → empty; resolver returns firstSeq.
	got, err := searchSeqByTime(5, 4, minute(3), fakeNextPresent())
	require.NoError(t, err)
	assert.Equal(t, uint64(5), got)
}

func TestSearchSeqByTime_OffsetFirstSeq(t *testing.T) {
	t.Parallel()
	// Stream whose first sequence is 100 (older messages aged out).
	at := fakeNextPresent(100, 101, 102, 103, 104)
	got, err := searchSeqByTime(100, 104, minute(102), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(102), got)

	// Target before the (new) first sequence clamps to firstSeq.
	got, err = searchSeqByTime(100, 104, minute(50), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(100), got)
}

func TestSearchSeqByTime_PropagatesError(t *testing.T) {
	t.Parallel()
	boom := errors.New("read failed")
	at := func(uint64) (uint64, time.Time, bool, error) { return 0, time.Time{}, false, boom }
	_, err := searchSeqByTime(1, 10, minute(5), at)
	require.ErrorIs(t, err, boom)
}

func TestSearchSeqByTime_AllHoles(t *testing.T) {
	t.Parallel()
	// Degenerate: no present message anywhere (shouldn't happen for a real
	// stream, but the search must not hang or panic); returns the "past end" default.
	at := func(uint64) (uint64, time.Time, bool, error) { return 0, time.Time{}, false, nil }
	got, err := searchSeqByTime(1, 10, minute(5), at)
	require.NoError(t, err)
	assert.Equal(t, uint64(11), got)
}

// fakeStream is a minimal jetstream.Stream implementing only GetMsg and
// CachedInfo; other methods panic if exercised (embedding keeps the mock small).
type fakeStream struct {
	jetstream.Stream
	info *jetstream.StreamInfo

	// present maps seq → publish time for messages that exist.
	present map[uint64]time.Time
	// sortedSeqs is present's keys in ascending order (for next-by-subject).
	sortedSeqs []uint64

	getErr error // when non-nil, every GetMsg returns this error
	calls  int
}

func newFakeStream(info *jetstream.StreamInfo, present map[uint64]time.Time) *fakeStream {
	seqs := make([]uint64, 0, len(present))
	for s := range present {
		seqs = append(seqs, s)
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	return &fakeStream{info: info, present: present, sortedSeqs: seqs}
}

func (f *fakeStream) CachedInfo() *jetstream.StreamInfo { return f.info }

func (f *fakeStream) GetMsg(ctx context.Context, seq uint64, opts ...jetstream.GetMsgOpt) (*jetstream.RawStreamMsg, error) {
	f.calls++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if f.getErr != nil {
		return nil, f.getErr
	}
	// We only ever call GetMsg with WithGetMsgSubject(">") in the resolver, so
	// emulate next-by-subject: first present message with seq >= requested.
	for _, s := range f.sortedSeqs {
		if s >= seq {
			return &jetstream.RawStreamMsg{Sequence: s, Time: f.present[s]}, nil
		}
	}
	return nil, jetstream.ErrMsgNotFound
}

func presentMap(present ...uint64) map[uint64]time.Time {
	m := make(map[uint64]time.Time, len(present))
	for _, s := range present {
		m[s] = timeBase.Add(time.Duration(s) * time.Minute)
	}
	return m
}

func streamInfo(firstSeq, lastSeq, msgs uint64) *jetstream.StreamInfo {
	return &jetstream.StreamInfo{
		State: jetstream.StreamState{FirstSeq: firstSeq, LastSeq: lastSeq, Msgs: msgs},
	}
}

// TestResolveSeqByTimeDirect_Wiring verifies StartTime → start sequence
// resolution end-to-end, including the next-by-subject probe across gaps.
func TestResolveSeqByTimeDirect_Wiring(t *testing.T) {
	t.Parallel()
	svc := &Client{}
	info := streamInfo(1, 10, 6)

	tests := []struct {
		name   string
		target time.Time
		want   uint64
	}{
		{"target in gap → next present", minute(7), 9},
		{"exact present", minute(6), 6},
		{"before first → firstSeq", minute(0), 1},
		{"after last → lastSeq+1", minute(10).Add(time.Minute), 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Present: 1,2,5,6,9,10 (3,4,7,8 deleted). Fresh per subtest so
			// parallel runs don't share the fakeStream's mutable call counter.
			stream := newFakeStream(info, presentMap(1, 2, 5, 6, 9, 10))
			got, err := svc.resolveSeqByTimeDirect(t.Context(), stream, info, tt.target)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestResolveSeqByTimeDirect_ContextCancellation verifies the resolver aborts
// promptly on an already-cancelled context, without retrying or scanning.
func TestResolveSeqByTimeDirect_ContextCancellation(t *testing.T) {
	t.Parallel()
	svc := &Client{}
	info := streamInfo(1, 1_000_000, 2)
	stream := newFakeStream(info, presentMap(1, 1_000_000))

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel before the search starts

	_, err := svc.resolveSeqByTimeDirect(ctx, stream, info, minute(500_000))
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, stream.calls, "no GetMsg should be issued once ctx is cancelled")
}

// TestNextMsgWithRetry_NotFoundTerminal confirms ErrMsgNotFound is terminal
// (not retried) so a "no message at/after seq" probe maps cleanly to a gap.
func TestNextMsgWithRetry_NotFoundTerminal(t *testing.T) {
	t.Parallel()
	svc := &Client{}
	stream := newFakeStream(streamInfo(1, 10, 0), presentMap()) // empty → always NotFound

	_, err := svc.nextMsgWithRetry(t.Context(), stream, 1)
	require.ErrorIs(t, err, jetstream.ErrMsgNotFound)
	assert.Equal(t, 1, stream.calls, "NotFound is terminal — no retries")
}
