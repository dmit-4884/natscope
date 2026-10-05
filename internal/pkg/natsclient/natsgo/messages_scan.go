// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"iter"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// fetchMethodConsumer reads messages through a short-lived consumer; any other method reads them one by one.
const fetchMethodConsumer = "consumer"

// Consumer scans pull adaptive batches of about scanFetchBytes, waiting up to scanFetchWait for each; a pull that
// keeps coming back empty while the consumer still has messages ends the scan with an error after scanEmptyFetches.
const (
	scanFetchFirst   = 32
	scanFetchMax     = 256
	scanFetchBytes   = 8 << 20
	scanFetchWait    = 5 * time.Second
	scanEmptyFetches = 3
)

// scanDirectBatch is how many sequences a direct scan reads per parallel round.
const scanDirectBatch = 200

// directGetHeaders are the headers the server adds to a direct read; they describe the read, not the message.
var directGetHeaders = []string{"Nats-Stream", "Nats-Sequence", "Nats-Subject", "Nats-Time-Stamp", "Nats-Last-Sequence", "Nats-Num-Pending"}

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

// scanViaConsumer streams the range through ephemeral consumers that the server filters by subject. A delivery that
// never reached this client shows as a gap in the consumer sequence; the scan then starts a new consumer after the
// last message it did receive, so a slow link costs time, never messages.
func (c *Client) scanViaConsumer(ctx context.Context, stream jetstream.Stream, opts entities.ScanOptions, yield func(*entities.Message, error) bool) {
	for from := opts.FromSeq; from <= opts.ToSeq; {
		next, ok := c.scanConsumerFrom(ctx, stream, opts, from, yield)
		if !ok {
			return
		}
		from = next
	}
}

// scanConsumerFrom reads from seq through one consumer; ok with the next sequence to read means the consumer lost a
// delivery and the scan should go on with a new one.
func (c *Client) scanConsumerFrom(
	ctx context.Context,
	stream jetstream.Stream,
	opts entities.ScanOptions,
	from uint64,
	yield func(*entities.Message, error) bool,
) (next uint64, ok bool) {
	cfg := jetstream.ConsumerConfig{
		Name:              browseConsumerPrefix + nats.NewInbox()[7:],
		DeliverPolicy:     jetstream.DeliverByStartSequencePolicy,
		OptStartSeq:       from,
		FilterSubject:     opts.SubjectFilter,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: browseConsumerInactiveThreshold,
		MemoryStorage:     true,
	}
	defer c.trackOwnConsumer(cfg.Name)()

	consumer, err := stream.CreateConsumer(ctx, cfg)
	if err != nil {
		yield(nil, wrapErr(err))
		return 0, false
	}
	defer func() {
		cleanupCtx, cancel := corecontext.ApplyTimeout(context.WithoutCancel(ctx), ephemeralCleanupTimeout)
		defer cancel()
		_ = stream.DeleteConsumer(cleanupCtx, cfg.Name) //nolint:errcheck // best-effort cleanup
	}()
	if info := consumer.CachedInfo(); info != nil && info.NumPending == 0 {
		return 0, false
	}

	var lastConsumerSeq, lastStreamSeq uint64
	var bytesRead, msgsRead int
	batchSize, empty := scanFetchFirst, 0
	for {
		batch, err := consumer.Fetch(batchSize, jetstream.FetchMaxWait(scanFetchWait))
		if err != nil {
			yield(nil, wrapErr(err))
			return 0, false
		}
		got := 0
		for msg := range batch.Messages() {
			got++
			meta, metaErr := msg.Metadata()
			if metaErr != nil || meta == nil {
				continue
			}
			if lastConsumerSeq > 0 && meta.Sequence.Consumer != lastConsumerSeq+1 {
				return lastStreamSeq + 1, true
			}
			lastConsumerSeq, lastStreamSeq = meta.Sequence.Consumer, meta.Sequence.Stream
			if meta.Sequence.Stream > opts.ToSeq {
				return 0, false
			}
			if !yield(scannedMessage(meta.Sequence.Stream, msg.Subject(), meta.Timestamp, msg.Data(), msg.Headers()), nil) {
				return 0, false
			}
			if meta.Sequence.Stream == opts.ToSeq || meta.NumPending == 0 {
				return 0, false
			}
			msgsRead++
			bytesRead += len(msg.Data())
		}
		if batchErr := batch.Error(); batchErr != nil && !errors.Is(batchErr, context.DeadlineExceeded) &&
			!errors.Is(batchErr, nats.ErrTimeout) {
			yield(nil, wrapErr(batchErr))
			return 0, false
		}
		if err := ctx.Err(); err != nil {
			yield(nil, wrapErr(err))
			return 0, false
		}
		if got > 0 {
			empty = 0
			if msgsRead > 0 {
				batchSize = min(scanFetchMax, max(1, scanFetchBytes/max(1, bytesRead/msgsRead)))
			}
			continue
		}
		if empty++; empty >= scanEmptyFetches {
			if async := c.takeAsyncError(scanFetchWait * scanEmptyFetches); async != nil {
				yield(nil, async)
			} else {
				yield(nil, wrapErr(fmt.Errorf("%w: the consumer has messages left but the server sent none", errs.ErrNATSTimeout)))
			}
			return 0, false
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
		if !yield(scannedRaw(msg), nil) {
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
		if ctxErr := ctx.Err(); ctxErr != nil {
			yield(nil, wrapErr(ctxErr))
			return
		}
		for _, seq := range seqs {
			msg, ok := found[seq]
			if !ok || (opts.SubjectFilter != "" && !natsutil.MatchSubject(opts.SubjectFilter, msg.Subject)) {
				continue
			}
			if !yield(scannedRaw(msg), nil) {
				return
			}
		}
		if end == opts.ToSeq {
			return
		}
		cursor = end + 1
		if len(found) > 0 {
			continue
		}
		next, err := c.nextMsgWithRetry(ctx, stream, cursor, ">")
		if errors.Is(err, jetstream.ErrMsgNotFound) {
			return
		}
		if err != nil {
			yield(nil, err)
			return
		}
		if next.Sequence > opts.ToSeq {
			return
		}
		if !yield(scannedRaw(next), nil) {
			return
		}
		cursor = next.Sequence + 1
	}
}

// scannedRaw converts a single-message read, without the headers the server adds to direct reads.
func scannedRaw(raw *jetstream.RawStreamMsg) *entities.Message {
	msg := toMessage(raw)
	for _, h := range directGetHeaders {
		delete(msg.Headers, h)
	}
	if len(msg.Headers) == 0 {
		msg.Headers = nil
	}
	return msg
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
