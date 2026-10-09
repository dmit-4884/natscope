// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// fakeClock advances by step every time a scanned message is read.
type fakeClock struct {
	now  time.Time
	step time.Duration
}

func (c *fakeClock) Now() time.Time { return c.now }

// fakeStream is a stream of sequences first..last minus deleted; every match-th sequence carries "needle".
type fakeStream struct {
	natssvc.StreamReader
	first, last uint64
	deleted     map[uint64]bool
	match       uint64
	clock       *fakeClock
	size        int
	onYield     func(*entities.Message)
}

func (f *fakeStream) has(seq uint64) bool { return seq >= f.first && seq <= f.last && !f.deleted[seq] }

func (f *fakeStream) matches(seq uint64) bool { return f.has(seq) && seq%f.match == 0 }

func (f *fakeStream) GetStreamInfo(context.Context, string, string) (*entities.StreamInfo, error) {
	return &entities.StreamInfo{State: &entities.StreamState{Msgs: f.last - f.first + 1, FirstSeq: f.first, LastSeq: f.last}}, nil
}

func (f *fakeStream) ScanMessages(ctx context.Context, _, _ string, opts entities.ScanOptions) iter.Seq2[*entities.Message, error] {
	return func(yield func(*entities.Message, error) bool) {
		for seq := opts.FromSeq; seq <= opts.ToSeq; seq++ {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			if !f.has(seq) {
				continue
			}
			f.clock.now = f.clock.now.Add(f.clock.step)
			payload := "hay"
			if seq%f.match == 0 {
				payload = "needle"
			}
			raw := []byte(payload + strings.Repeat(" ", max(f.size-len(payload), 0)))
			msg := &entities.Message{Sequence: seq, Subject: "s", DataBase64: base64.StdEncoding.EncodeToString(raw), DataSize: len(raw)}
			if f.onYield != nil {
				f.onYield(msg)
			}
			if !yield(msg, nil) {
				return
			}
		}
	}
}

type noCodec struct{ protosvc.Codec }

func (noCodec) DecodeMessages(context.Context, []*entities.Message, bool) {}

type noSettings struct{ settingssvc.Service }

func (noSettings) Get(context.Context) (*entities.UserSettings, error) {
	return nil, errors.New("no settings")
}

var errStopped = errors.New("stopped by the client")

type runResult struct {
	matches []uint64
	resume  uint64
	done    *entities.MessageSearchDone
	stopped bool
}

// runSearch runs one search; stopAfter > 0 makes the client hang up at that progress event, as Stop does.
func runSearch(t *testing.T, s *Service, req entities.MessageSearchRequest, stopAfter int) runResult {
	t.Helper()
	var res runResult
	progressEvents := 0
	err := s.Search(t.Context(), &req, func(ev *entities.MessageSearchEvent) error {
		switch {
		case ev.Matches != nil:
			for _, m := range ev.Matches {
				res.matches = append(res.matches, m.Sequence)
			}
		case ev.Progress != nil:
			res.resume = ev.Progress.ResumeSeq
			progressEvents++
			if stopAfter > 0 && progressEvents == stopAfter {
				return errStopped
			}
		case ev.Done != nil:
			res.done = ev.Done
		}
		return nil
	})
	if errors.Is(err, errStopped) {
		res.stopped = true
		return res
	}
	require.NoError(t, err)
	return res
}

// searchAll follows the client through Stop and "Search further" until the search completes.
func searchAll(t *testing.T, s *Service, req entities.MessageSearchRequest, stopAt func(run int) int) []uint64 {
	t.Helper()
	var all []uint64
	for run := 0; run < 1000; run++ {
		res := runSearch(t, s, req, stopAt(run))
		all = append(all, res.matches...)
		var next uint64
		switch {
		case res.stopped:
			next = res.resume
		case res.done.NextSeq != 0:
			next = res.done.NextSeq
		default:
			return all
		}
		require.NotZero(t, next, "run %d stopped without a place to resume", run)
		req.CursorSeq = new(next)
	}
	t.Fatal("the search never completed")
	return nil
}

func expectedMatches(f *fakeStream, backward bool) []uint64 {
	var want []uint64
	for seq := f.first; seq <= f.last; seq++ {
		if f.matches(seq) {
			want = append(want, seq)
		}
	}
	if backward {
		slices.Reverse(want)
	}
	return want
}

func newFakeSearch(t *testing.T, f *fakeStream) *Service {
	t.Helper()
	clock := f.clock
	previous := searchClock
	searchClock = clock.Now
	t.Cleanup(func() { searchClock = previous })
	return New(f, noCodec{}, noSettings{})
}

func TestSearchStopAndSearchFurtherNeitherSkipsNorRepeats(t *testing.T) {
	deleted := map[uint64]bool{}
	for seq := uint64(1500); seq < 1700; seq++ {
		deleted[seq] = true
	}
	for _, dir := range []string{"backward", "forward"} {
		for stopAt := 1; stopAt <= 12; stopAt++ {
			t.Run(fmt.Sprintf("%s stop at progress %d", dir, stopAt), func(t *testing.T) {
				f := &fakeStream{first: 101, last: 6000, deleted: deleted, match: 7, clock: &fakeClock{now: time.Unix(0, 0), step: 40 * time.Millisecond}}
				s := newFakeSearch(t, f)

				got := searchAll(t, s, entities.MessageSearchRequest{Direction: dir, Text: "needle"}, func(run int) int {
					if run%2 == 0 {
						return stopAt
					}
					return 0
				})

				assert.Equal(t, expectedMatches(f, dir == "backward"), got)
			})
		}
	}
}

func TestSearchBudgetsBoundEveryRun(t *testing.T) {
	for _, dir := range []string{"backward", "forward"} {
		t.Run(dir+" time", func(t *testing.T) {
			f := &fakeStream{first: 1, last: 400_000, match: 997, clock: &fakeClock{now: time.Unix(0, 0), step: 1500 * time.Microsecond}}
			s := newFakeSearch(t, f)
			started := f.clock.now

			res := runSearch(t, s, entities.MessageSearchRequest{Direction: dir, Text: "needle"}, 0)

			assert.Equal(t, entities.SearchStopTimeLimit, res.done.Reason)
			assert.LessOrEqual(t, f.clock.now.Sub(started), searchMaxDuration+2*time.Second)
		})
		t.Run(dir+" scanned", func(t *testing.T) {
			f := &fakeStream{first: 1, last: 400_000, match: 997, clock: &fakeClock{now: time.Unix(0, 0), step: time.Microsecond}}
			s := newFakeSearch(t, f)

			res := runSearch(t, s, entities.MessageSearchRequest{Direction: dir, Text: "needle"}, 0)

			assert.Equal(t, entities.SearchStopScanLimit, res.done.Reason)
			assert.LessOrEqual(t, res.done.Scanned, uint64(searchMaxScanned))
		})
		t.Run(dir+" continues to the end without gaps", func(t *testing.T) {
			f := &fakeStream{first: 1, last: 250_000, match: 997, clock: &fakeClock{now: time.Unix(0, 0), step: 200 * time.Microsecond}}
			s := newFakeSearch(t, f)

			got := searchAll(t, s, entities.MessageSearchRequest{Direction: dir, Text: "needle"}, func(int) int { return 0 })

			assert.Equal(t, expectedMatches(f, dir == "backward"), got)
		})
	}
}

func TestSearchBackwardSlowFirstWindowKeepsTheTimeBudget(t *testing.T) {
	slow := func() *fakeStream {
		return &fakeStream{first: 1, last: 3000, match: 7, clock: &fakeClock{now: time.Unix(0, 0), step: 100 * time.Millisecond}}
	}
	t.Run("one run", func(t *testing.T) {
		f := slow()
		s := newFakeSearch(t, f)
		started := f.clock.now

		res := runSearch(t, s, entities.MessageSearchRequest{Direction: "backward", Text: "needle"}, 0)

		assert.Equal(t, entities.SearchStopTimeLimit, res.done.Reason)
		assert.LessOrEqual(t, f.clock.now.Sub(started), searchMaxDuration+2*time.Second)
		assert.NotEmpty(t, res.matches)
	})
	t.Run("continues to the end without gaps", func(t *testing.T) {
		f := slow()
		s := newFakeSearch(t, f)

		got := searchAll(t, s, entities.MessageSearchRequest{Direction: "backward", Text: "needle"}, func(int) int { return 0 })

		assert.Equal(t, expectedMatches(f, true), got)
	})
}

func TestSearchProgressNamesWhereToResumeFromTheStart(t *testing.T) {
	f := &fakeStream{first: 1, last: 5000, match: 7, clock: &fakeClock{now: time.Unix(0, 0), step: time.Millisecond}}
	s := newFakeSearch(t, f)

	back := runSearch(t, s, entities.MessageSearchRequest{Direction: "backward", Text: "needle", CursorSeq: new(uint64(4000))}, 1)
	assert.Equal(t, uint64(4000), back.resume)

	fwd := runSearch(t, s, entities.MessageSearchRequest{Direction: "forward", Text: "needle", CursorSeq: new(uint64(1200))}, 1)
	assert.Equal(t, uint64(1200), fwd.resume)
}

func TestSearchBackwardDoesNotHoldWholePayloadsOfAWindow(t *testing.T) {
	const limit = 1024
	type held struct {
		msg      *entities.Message
		original *byte
	}
	var seen []held
	var oversized []uint64
	f := &fakeStream{first: 1, last: 600, match: 1, size: 64 << 10, clock: &fakeClock{now: time.Unix(0, 0), step: time.Microsecond}}
	f.onYield = func(next *entities.Message) {
		for _, h := range seen {
			if len(h.msg.DataBase64) > limit*2 || unsafe.StringData(h.msg.DataBase64) == h.original {
				oversized = append(oversized, h.msg.Sequence)
			}
		}
		seen = append(seen, held{msg: next, original: unsafe.StringData(next.DataBase64)})
	}
	s := newFakeSearch(t, f)

	res := runSearch(t, s, entities.MessageSearchRequest{Direction: "backward", Text: "needle", MaxPayloadBytes: new(int32(limit))}, 0)

	assert.Len(t, res.matches, searchMaxMatches)
	assert.Empty(t, oversized, "a match waiting for its window keeps only the trimmed payload")
}

type countingCodec struct {
	protosvc.Codec
	calls map[string]int
}

func (c *countingCodec) DecodeMessages(_ context.Context, msgs []*entities.Message, _ bool) {
	for _, m := range msgs {
		c.calls[m.Subject]++
		if m.Subject == "mapped" {
			m.Decoded = []byte(`{"field":"value"}`)
		}
	}
}

func TestSearchTriesAnUnmappedSubjectOnce(t *testing.T) {
	f := &fakeStream{first: 1, last: 50, match: 1000, clock: &fakeClock{now: time.Unix(0, 0), step: time.Millisecond}}
	codec := &countingCodec{calls: map[string]int{}}
	previous := searchClock
	searchClock = f.clock.Now
	t.Cleanup(func() { searchClock = previous })
	s := New(f, codec, noSettings{})

	res := runSearch(t, s, entities.MessageSearchRequest{Direction: "forward", Text: "absent"}, 0)

	assert.Empty(t, res.matches)
	assert.Equal(t, 1, codec.calls["s"], "a subject that decodes to nothing is not tried again")
}
