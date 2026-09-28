// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/retry"
	"github.com/altessa-s/go-atlas/core/runtime/concurrency"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// Tunables for parallel-GetMsg fetch sizing.
const (
	// denseFetchMultiplier over-fetches (limit+1)*N sequences on dense streams and
	// filters empties.
	denseFetchMultiplier = 2

	// sparseDensityCushion is the 50% overshoot on the gap-corrected estimate,
	// absorbing non-uniform deletions.
	sparseDensityCushion = 1.5
)

// getMessagesParallel fetches via parallel direct GetMsg calls (no consumers);
// best for dense streams without subject filters.
func (c *Client) getMessagesParallel(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	startSeq uint64,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	if info.State.Msgs == 0 {
		return &entities.MessagesResponse{
			Messages: []*entities.Message{},
			HasMore:  false,
		}, nil
	}

	if startSeq == 0 {
		if direction == DefaultDirection {
			startSeq = info.State.LastSeq
		} else {
			startSeq = info.State.FirstSeq
		}
	} else {
		// Clamp an out-of-range startSeq into the live range instead of walking
		// entirely below FirstSeq (e.g. after a purge — forward pagination would
		// query only already-purged sequences, find nothing, and report
		// hasMore=true with nextSeq=1, looping forever, QA-019) or above LastSeq
		// (backward with a huge/overflowed startSeq returning an empty page
		// instead of the newest messages, QA-120).
		startSeq = min(max(startSeq, info.State.FirstSeq), info.State.LastSeq)
	}

	t0 := time.Now()

	// Size the fetch for gaps: dense → limit*denseFetchMultiplier, sparse → widen
	// proportionally up to DefaultSearchRange.
	needed := limit + 1
	fetchCount := needed * denseFetchMultiplier
	totalRange := info.State.LastSeq - info.State.FirstSeq + 1
	if totalRange > 0 && info.State.Msgs > 0 && info.State.Msgs < totalRange {
		density := float64(info.State.Msgs) / float64(totalRange)
		if density > 0 {
			estimated := int(float64(needed) / density * sparseDensityCushion)
			fetchCount = min(max(fetchCount, estimated), DefaultSearchRange)
		}
	}
	// Clamp to message count: heavy retention pruning makes totalRange huge while
	// Msgs stays small, else we'd fire thousands of not-found GetMsg calls.
	if info.State.Msgs > 0 && uint64(fetchCount) > info.State.Msgs {
		fetchCount = int(info.State.Msgs) + 1
	}

	seqsToFetch := buildSequenceList(startSeq, info, direction, fetchCount)
	if len(seqsToFetch) == 0 {
		return &entities.MessagesResponse{
			Messages: []*entities.Message{},
			HasMore:  false,
		}, nil
	}

	msgMap, err := c.fetchMessagesParallel(ctx, stream, seqsToFetch)
	if err != nil {
		return nil, wrapErr(err)
	}

	var messages []*entities.Message //nolint:prealloc
	var lastProcessedSeq uint64

	for _, seq := range seqsToFetch {
		if len(messages) >= needed {
			break
		}

		msg, ok := msgMap[seq]
		if !ok {
			continue
		}

		messages = append(messages, toMessage(msg))
		lastProcessedSeq = seq
	}

	// hasMore reflects whether the +1 probe message was actually collected —
	// deriving it only from the early-break flag above missed the case where
	// the probe was the very last entry in seqsToFetch (a stream of exactly
	// limit+1 messages), silently losing that last message (QA-018).
	hasMore := len(messages) > limit

	// Trim the extra +1 message used for hasMore detection.
	if hasMore {
		messages = messages[:limit]
	}

	// Cursor from the last kept message; if the window came back empty (an
	// interior gap wider than the fetch window, or nothing but holes to the
	// stream boundary), resume from the end of the queried window instead of
	// falling back to 0 — which wraps nextSeq to 1 and loops forever (QA-019).
	cursorSeq := lastProcessedSeq
	switch {
	case len(messages) > 0:
		cursorSeq = messages[len(messages)-1].Sequence
	case len(seqsToFetch) > 0:
		cursorSeq = seqsToFetch[len(seqsToFetch)-1]
	}

	// The fetch window may have stopped short of the stream boundary (sized by
	// fetchCount, not FirstSeq/LastSeq) without collecting `needed` messages —
	// there can still be more beyond it.
	if !hasMore {
		hasMore = hasMoreMessages(direction, cursorSeq, info)
	}

	nextSeq := calculateNextSeq(hasMore, direction, cursorSeq, info)

	c.logger.Debug("getMessagesParallel OK",
		slogx.String("stream", info.Config.Name),
		slogx.String("direction", direction),
		slogx.String("elapsed", time.Since(t0).String()),
		slogx.Int("fetched_seqs", len(seqsToFetch)),
		slogx.Int("found_msgs", len(msgMap)),
		slogx.Int("returned", len(messages)))

	return &entities.MessagesResponse{
		Messages: messages,
		HasMore:  hasMore,
		NextSeq:  nextSeq,
	}, nil
}

// fetchParallelTimeout is the fallback batch deadline when the caller's ctx has
// none, so a wedged server can't hang the response for minutes.
const fetchParallelTimeout = 30 * time.Second

// transientFetchRetries is the per-seq retry budget for transient errors and
// per-attempt timeouts (~15s on a wedged path, inside the 30s budget).
const transientFetchRetries = 3

// transientFetchBackoff is the inter-retry wait — short enough for the budget,
// long enough to let reconnect recover.
const transientFetchBackoff = 250 * time.Millisecond

// perAttemptTimeout caps each GetMsg so the first call can't wedge the full
// deadline after a reconnect (reply routing briefly desynced), starving retries.
const perAttemptTimeout = 5 * time.Second

// isTransientFetchError reports whether a GetMsg failure is worth retrying
// (NATS "later" errors + per-attempt deadline); parent-ctx cancel handled separately.
func isTransientFetchError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, nats.ErrConnectionClosed) ||
		errors.Is(err, nats.ErrTimeout) ||
		errors.Is(err, nats.ErrNoResponders) ||
		errors.Is(err, nats.ErrDisconnected) {
		return true
	}
	return false
}

// fetchMessagesParallel fans seqs across an I/O-bound worker pool; the first
// hard failure discards the partial result, holes (ErrMsgNotFound) are skipped.
func (c *Client) fetchMessagesParallel(
	ctx context.Context,
	stream jetstream.Stream,
	seqs []uint64,
) (map[uint64]*jetstream.RawStreamMsg, error) {
	ctx, cancel := corecontext.WithDefault(ctx, fetchParallelTimeout)
	defer cancel()

	var mu sync.Mutex
	msgMap := make(map[uint64]*jetstream.RawStreamMsg, len(seqs))

	// stopOnError cancels the batch on the first hard failure; holes
	// (ErrMsgNotFound under retention pruning) are skipped, not aborted.
	err := concurrency.Process(ctx, seqs,
		func(ctx context.Context, seq uint64) error {
			msg, err := c.getMsgWithRetry(ctx, stream, seq)
			if err != nil {
				if errors.Is(err, jetstream.ErrMsgNotFound) {
					return nil
				}
				return err
			}
			mu.Lock()
			msgMap[seq] = msg
			mu.Unlock()
			return nil
		},
		concurrency.WithConcurrency[uint64](min(optimalFetchConcurrency(), len(seqs))),
		concurrency.WithStopOnError[uint64](),
	)

	// A parent-context cancel returns whatever was collected; any other cause
	// discards the partial result.
	if err != nil && !errors.Is(err, context.Canceled) {
		c.logger.WarnContext(ctx, "fetchMessagesParallel: fetch failed",
			slog.Int("total", len(seqs)),
			slog.Int("fetched_before_abort", len(msgMap)),
			slog.String("error", err.Error()))
		return nil, err
	}

	return msgMap, nil
}

// getMsgWithRetry retries GetMsg on transient errors and per-attempt timeouts;
// parent-ctx cancel aborts immediately.
func (c *Client) getMsgWithRetry(
	ctx context.Context,
	stream jetstream.Stream,
	seq uint64,
) (*jetstream.RawStreamMsg, error) {
	var msg *jetstream.RawStreamMsg
	err := retry.Do(ctx, func(ctx context.Context) error {
		attemptCtx, cancel := corecontext.WithMaxTimeout(ctx, perAttemptTimeout)
		defer cancel()
		got, err := stream.GetMsg(attemptCtx, seq)
		if err != nil {
			return err
		}
		msg = got
		return nil
	},
		retry.WithMaxAttempts(transientFetchRetries),
		retry.WithNextDelay(func(int, error) time.Duration { return transientFetchBackoff }),
		retry.WithShouldRetry(fetchRetryable(ctx)),
	)
	if err != nil {
		return nil, wrapErr(err)
	}
	return msg, nil
}

// fetchRetryable retries per-attempt timeouts and transient NATS errors; a
// parent-ctx cancel or a missing message (retention hole) is terminal.
func fetchRetryable(parent context.Context) retry.ShouldRetryFunc {
	return func(err error) bool {
		if parent.Err() != nil || errors.Is(err, jetstream.ErrMsgNotFound) {
			return false
		}
		return errors.Is(err, context.DeadlineExceeded) || isTransientFetchError(err)
	}
}
