// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/retry"

	"github.com/dmit-4884/natscope/internal/errs"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// nextPresentFunc returns the first present message at/after seq in one
// round-trip; found=false means seq is past the last live message.
type nextPresentFunc func(seq uint64) (foundSeq uint64, t time.Time, found bool, err error)

// searchSeqByTime binary-searches for the smallest seq with timestamp >=
// target in O(log N); only approximate on sourced/mirror streams — prefer
// resolveSeqByTimeConsumer there.
func searchSeqByTime(firstSeq, lastSeq uint64, target time.Time, nextPresent nextPresentFunc) (uint64, error) {
	if lastSeq < firstSeq {
		return firstSeq, nil
	}

	lo, hi := firstSeq, lastSeq
	result := lastSeq + 1 // default: nothing is at/after target

	for lo <= hi {
		mid := lo + (hi-lo)/2 //nolint:mnd // binary-search midpoint

		seq, t, found, err := nextPresent(mid)
		if err != nil {
			return 0, err
		}
		if !found || seq > hi {
			// No present message in [mid, hi] — any qualifying message is below mid.
			if mid == 0 {
				break
			}
			hi = mid - 1
			continue
		}

		if !t.Before(target) { // t >= target: candidate, look for an earlier one
			result = seq
			if seq == 0 {
				break
			}
			hi = seq - 1
		} else { // t < target: go higher
			lo = seq + 1
		}
	}

	return result, nil
}

// resolveSeqByTime maps a target time to the paging start sequence (0 for
// empty streams); bounded by DefaultTimeout so a jump-to-time click can't hang.
func (c *Client) resolveSeqByTime(
	ctx context.Context,
	streamName string,
	fetchMethod string,
	target time.Time,
) (uint64, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, DefaultTimeout)
	defer cancel()

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}
	info := stream.CachedInfo()
	if info == nil || info.State.LastSeq == 0 {
		return 0, nil
	}

	if fetchMethod == "consumer" {
		return c.resolveSeqByTimeConsumer(ctx, stream, info, target)
	}
	return c.resolveSeqByTimeDirect(ctx, stream, info, target)
}

// resolveSeqByTimeDirect binary-searches via next-by-subject GetMsg(">") reads
// (fetch_method "direct", no create-consumer perm); jumps gaps per probe.
func (c *Client) resolveSeqByTimeDirect(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	target time.Time,
) (uint64, error) {
	nextPresent := func(seq uint64) (uint64, time.Time, bool, error) {
		msg, err := c.nextMsgWithRetry(ctx, stream, seq)
		if err != nil {
			// No present message at/after seq — treat as past the end.
			if errors.Is(err, jetstream.ErrMsgNotFound) {
				return 0, time.Time{}, false, nil
			}
			return 0, time.Time{}, false, err
		}
		return msg.Sequence, msg.Time, true, nil
	}

	seq, err := searchSeqByTime(info.State.FirstSeq, info.State.LastSeq, target, nextPresent)
	if err != nil {
		return 0, wrapErr(err)
	}
	return seq, nil
}

// nextMsgWithRetry is a next-by-subject GetMsg with getMsgWithRetry's transient
// retry behavior (ErrMsgNotFound terminal, parent-ctx cancel aborts).
func (c *Client) nextMsgWithRetry(
	ctx context.Context,
	stream jetstream.Stream,
	seq uint64,
) (*jetstream.RawStreamMsg, error) {
	var msg *jetstream.RawStreamMsg
	err := retry.Do(ctx, func(ctx context.Context) error {
		attemptCtx, cancel := corecontext.WithMaxTimeout(ctx, perAttemptTimeout)
		defer cancel()
		got, err := stream.GetMsg(attemptCtx, seq, jetstream.WithGetMsgSubject(">"))
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

// resolveSeqByTimeConsumer uses an ephemeral DeliverByStartTime consumer for
// server-side, mirror-safe resolution; returns lastSeq+1 when target is future.
func (c *Client) resolveSeqByTimeConsumer(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	target time.Time,
) (uint64, error) {
	if info.Config.Retention == jetstream.WorkQueuePolicy {
		// AckNone consumer would drain a WorkQueue stream just by reading it.
		return 0, errs.ErrWorkQueueConsumerNotAllowed
	}

	startTime := target
	ephCfg := jetstream.ConsumerConfig{
		DeliverPolicy:     jetstream.DeliverByStartTimePolicy,
		OptStartTime:      &startTime,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: browseConsumerInactiveThreshold,
		Name:              fmt.Sprintf("natscope-timeres-%s", nats.NewInbox()[7:]),
	}

	consumer, err := stream.CreateConsumer(ctx, ephCfg)
	if err != nil {
		return 0, wrapErr(err)
	}
	defer func() {
		_ = stream.DeleteConsumer(ctx, ephCfg.Name) //nolint:errcheck // best-effort cleanup
	}()

	batch, err := consumer.FetchNoWait(1)
	if err != nil {
		return 0, wrapErr(coreerrs.Wrap(err, "fetch by time"))
	}

	for msg := range batch.Messages() {
		meta, metaErr := msg.Metadata()
		if metaErr != nil || meta == nil {
			continue
		}
		return meta.Sequence.Stream, nil
	}
	// A plain fetch deadline means no message at/after target; the watched
	// client already recovered any out-of-band permissions violation.
	if batchErr := batch.Error(); batchErr != nil && !errors.Is(batchErr, context.DeadlineExceeded) {
		return 0, wrapErr(batchErr)
	}

	// Nothing at/after target — page from just past the end (empty result).
	return info.State.LastSeq + 1, nil
}
