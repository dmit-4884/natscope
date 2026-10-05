// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"iter"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// fetchMethodConsumer reads messages through a short-lived consumer; any other method reads them one by one.
const fetchMethodConsumer = "consumer"

// scanFetchBatch is how many messages a consumer scan pulls per request.
const scanFetchBatch = 256

// scanDirectBatch is how many sequences a direct scan reads per parallel round.
const scanDirectBatch = 200

// ScanMessages reads the stored messages from opts.FromSeq to opts.ToSeq oldest first. It reads through a
// short-lived consumer, or with single-message reads for the "direct" fetch method and for work queues, which a
// consumer would drain. Stopping the iteration releases the consumer.
func (c *Client) ScanMessages(ctx context.Context, streamName string, opts entities.ScanOptions) iter.Seq2[*entities.Message, error] {
	return func(yield func(*entities.Message, error) bool) {
		if err := validateNATSNameLength("stream name", streamName); err != nil {
			yield(nil, wrapErr(err))
			return
		}
		if err := validateNATSSubjectLength("subject filter", opts.SubjectFilter); err != nil {
			yield(nil, wrapErr(err))
			return
		}
		if opts.FromSeq == 0 || opts.ToSeq < opts.FromSeq {
			return
		}

		stream, err := c.jetStream.Stream(ctx, streamName)
		if err != nil {
			yield(nil, wrapErr(err))
			return
		}

		info := stream.CachedInfo()
		workQueue := info != nil && info.Config.Retention == jetstream.WorkQueuePolicy
		switch {
		case opts.FetchMethod == fetchMethodConsumer && !workQueue:
			c.scanViaConsumer(ctx, stream, opts, yield)
		case opts.SubjectFilter != "":
			c.scanBySubject(ctx, stream, opts, yield)
		default:
			c.scanDirect(ctx, stream, opts, yield)
		}
	}
}

// scanViaConsumer streams the range through an ephemeral consumer that the server filters by subject.
func (c *Client) scanViaConsumer(ctx context.Context, stream jetstream.Stream, opts entities.ScanOptions, yield func(*entities.Message, error) bool) {
	cfg := jetstream.ConsumerConfig{
		Name:              browseConsumerPrefix + nats.NewInbox()[7:],
		DeliverPolicy:     jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:       opts.FromSeq,
		FilterSubject:     opts.SubjectFilter,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: browseConsumerInactiveThreshold,
		MemoryStorage:     true,
	}
	defer c.trackOwnConsumer(cfg.Name)()

	consumer, err := stream.CreateConsumer(ctx, cfg)
	if err != nil {
		yield(nil, wrapErr(err))
		return
	}
	defer func() {
		cleanupCtx, cancel := corecontext.ApplyTimeout(context.WithoutCancel(ctx), ephemeralCleanupTimeout)
		defer cancel()
		_ = stream.DeleteConsumer(cleanupCtx, cfg.Name) //nolint:errcheck // best-effort cleanup
	}()

	for {
		batch, err := consumer.FetchNoWait(scanFetchBatch)
		if err != nil {
			yield(nil, wrapErr(err))
			return
		}
		got := 0
		for msg := range batch.Messages() {
			got++
			meta, metaErr := msg.Metadata()
			if metaErr != nil || meta == nil {
				continue
			}
			if meta.Sequence.Stream > opts.ToSeq {
				return
			}
			if !yield(scannedMessage(meta.Sequence.Stream, msg.Subject(), meta.Timestamp, msg.Data(), msg.Headers()), nil) {
				return
			}
		}
		if batchErr := batch.Error(); batchErr != nil && !errors.Is(batchErr, context.DeadlineExceeded) {
			yield(nil, wrapErr(batchErr))
			return
		}
		if got == 0 {
			return
		}
		if err := ctx.Err(); err != nil {
			yield(nil, wrapErr(err))
			return
		}
	}
}

// scanBySubject walks the range with next-by-subject reads, so the server skips other subjects.
func (c *Client) scanBySubject(ctx context.Context, stream jetstream.Stream, opts entities.ScanOptions, yield func(*entities.Message, error) bool) {
	for cursor := opts.FromSeq; cursor <= opts.ToSeq; {
		msg, err := c.nextMsgWithRetry(ctx, stream, cursor, opts.SubjectFilter)
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return
		}
		if err != nil {
			yield(nil, err)
			return
		}
		if msg.Sequence > opts.ToSeq {
			return
		}
		if !yield(toMessage(msg), nil) {
			return
		}
		cursor = msg.Sequence + 1
	}
}

// scanDirect reads every sequence of the range in parallel rounds, skipping deleted ones.
func (c *Client) scanDirect(ctx context.Context, stream jetstream.Stream, opts entities.ScanOptions, yield func(*entities.Message, error) bool) {
	for cursor := opts.FromSeq; cursor <= opts.ToSeq; {
		end := min(opts.ToSeq, cursor+scanDirectBatch-1)
		seqs := make([]uint64, 0, end-cursor+1)
		for seq := cursor; seq <= end; seq++ {
			seqs = append(seqs, seq)
		}
		found, err := c.fetchMessagesParallel(ctx, stream, seqs)
		if err != nil {
			yield(nil, wrapErr(err))
			return
		}
		if err := ctx.Err(); err != nil {
			yield(nil, wrapErr(err))
			return
		}
		for _, seq := range seqs {
			msg, ok := found[seq]
			if !ok || (opts.SubjectFilter != "" && !natsutil.MatchSubject(opts.SubjectFilter, msg.Subject)) {
				continue
			}
			if !yield(toMessage(msg), nil) {
				return
			}
		}
		if end == opts.ToSeq {
			return
		}
		cursor = end + 1
	}
}

// scannedMessage builds the entity a scan yields from a delivered consumer message.
func scannedMessage(seq uint64, subject string, at time.Time, data []byte, header nats.Header) *entities.Message {
	var headers map[string]string
	if len(header) > 0 {
		headers = make(map[string]string, len(header))
		for k, v := range header {
			headers[k] = strings.Join(v, ", ")
		}
	}
	return &entities.Message{
		Sequence:    seq,
		Subject:     subject,
		Timestamp:   at,
		DataBase64:  base64.StdEncoding.EncodeToString(data),
		DataSize:    len(data),
		ContentType: entities.DetectContentType(data),
		Headers:     headers,
	}
}

// SeqAtTime returns the first sequence stored at or after t, or the last sequence plus one when none is.
func (c *Client) SeqAtTime(ctx context.Context, streamName, fetchMethod string, t time.Time) (uint64, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return 0, wrapErr(err)
	}
	return c.resolveSeqByTime(ctx, streamName, fetchMethod, t)
}
