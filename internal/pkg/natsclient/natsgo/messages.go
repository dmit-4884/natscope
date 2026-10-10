// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// GetMessages picks a fetch strategy: "consumer" → ephemeral consumer
// (sparse/filtered), subject filter → direct GetMsg, else parallel GetMsg.
func (c *Client) GetMessages(
	ctx context.Context,
	streamName string,
	opts entities.GetMessagesOptions,
) (*entities.MessagesResponse, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSSubjectLength("subject filter", opts.SubjectFilter); err != nil {
		return nil, wrapErr(err)
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = DefaultMessageLimit
	}
	limit = min(limit, DefaultMaxMessageLimit)

	direction := opts.Direction
	if direction == "" {
		direction = DefaultDirection
	}

	// Jump-to-time: resolve target time to a start sequence, then page as a
	// start_seq request would.
	if opts.StartTime != nil {
		seq, resolveErr := c.resolveSeqByTime(ctx, streamName, opts.FetchMethod, *opts.StartTime)
		if resolveErr != nil {
			return nil, resolveErr
		}
		opts.StartSeq = seq
	}

	if opts.FetchMethod == fetchMethodConsumer {
		return c.getMessagesViaConsumer(ctx, streamName, opts.SubjectFilter, opts.StartSeq, limit, direction)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get stream"))
	}

	return c.getMessagesByGet(ctx, stream, stream.CachedInfo(), opts.SubjectFilter, opts.StartSeq, limit, direction)
}

// getMessagesByGet reads a page with per-sequence gets, which never consume a message.
func (c *Client) getMessagesByGet(
	ctx context.Context,
	stream jetstream.Stream,
	info *jetstream.StreamInfo,
	subjectFilter string,
	startSeq uint64,
	limit int,
	direction string,
) (*entities.MessagesResponse, error) {
	if subjectFilter != "" {
		return c.getMessagesWithSubjectFilter(ctx, stream, info, subjectFilter, startSeq, limit, direction)
	}
	return c.getMessagesParallel(ctx, stream, info, startSeq, limit, direction)
}

// GetMessage fetches a single message by sequence number.
func (c *Client) GetMessage(ctx context.Context, streamName string, sequence uint64) (*entities.Message, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get stream"))
	}

	msg, err := stream.GetMsg(ctx, sequence)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get message"))
	}

	return toMessage(msg), nil
}

// Publish sends a core NATS message and flushes it.
func (c *Client) Publish(ctx context.Context, subject string, data []byte, headers map[string]string) error {
	if err := validateNATSSubjectLength("subject", subject); err != nil {
		return wrapErr(err)
	}

	msg := &nats.Msg{Subject: subject, Data: data}
	if len(headers) > 0 {
		msg.Header = make(nats.Header, len(headers))
		for k, v := range headers {
			msg.Header.Set(k, v)
		}
	}

	ctx, cancel := corecontext.ApplyTimeout(ctx, c.defaultTimeout)
	defer cancel()

	before := c.conn.LastError()
	err := c.permWatch.Watch(ctx, []string{subject}, func(ctx context.Context) error {
		if err := c.conn.PublishMsg(msg); err != nil {
			return err
		}
		if err := c.conn.FlushWithContext(ctx); err != nil {
			return err
		}
		if last := c.conn.LastError(); last != nil && last != before { //nolint:errorlint // identity check
			if v, ok := ParsePermissionViolation(last); ok && v.Operation == violatedPublish && v.Subject == subject {
				return last
			}
		}
		return nil
	})
	return wrapErr(err)
}

// PublishToStream publishes a message to a JetStream stream after checking that a stream captures subject.
func (c *Client) PublishToStream(
	ctx context.Context,
	subject string,
	data []byte,
	headers map[string]string,
) (*entities.PubAck, error) {
	if err := validateNATSSubjectLength("subject", subject); err != nil {
		return nil, wrapErr(err)
	}
	if err := c.requireFeatures(ctx, headerFeatures(headers)...); err != nil {
		return nil, err
	}
	if _, err := c.jetStream.StreamNameBySubject(ctx, subject); err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "resolve stream for subject"))
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}

	if len(headers) > 0 {
		msg.Header = make(nats.Header)
		for k, v := range headers {
			msg.Header.Set(k, v)
		}
	}

	ack, err := c.jetStream.PublishMsg(ctx, msg)
	if err != nil {
		return nil, wrapErr(err)
	}

	return &entities.PubAck{
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Domain:    ack.Domain,
		Duplicate: ack.Duplicate,
		Value:     ack.Value,
	}, nil
}
