// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"

	"github.com/nats-io/nats.go"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"
)

// GetMessages picks a fetch strategy: "consumer" → ephemeral consumer
// (sparse/filtered), subject filter → direct GetMsg, else parallel GetMsg.
func (c *Client) GetMessages(
	ctx context.Context,
	streamName string,
	opts entities.GetMessagesOptions,
) (*entities.MessagesResponse, error) {
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

	if opts.FetchMethod == "consumer" {
		return c.getMessagesViaConsumer(ctx, streamName, opts.SubjectFilter, opts.StartSeq, limit, direction)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get stream"))
	}

	info := stream.CachedInfo()

	if opts.SubjectFilter != "" {
		return c.getMessagesWithSubjectFilter(ctx, stream, info, opts.SubjectFilter, opts.StartSeq, limit, direction)
	}

	return c.getMessagesParallel(ctx, stream, info, opts.StartSeq, limit, direction)
}

// GetMessage fetches a single message by sequence number.
func (c *Client) GetMessage(ctx context.Context, streamName string, sequence uint64) (*entities.Message, error) {
	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get stream"))
	}

	msg, err := stream.GetMsg(ctx, sequence)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get message"))
	}

	return toMessageWithHex(msg), nil
}

// PublishToStream publishes a message to a JetStream stream.
func (c *Client) PublishToStream(
	ctx context.Context,
	subject string,
	data []byte,
	headers map[string]string,
) (*entities.PubAck, error) {
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
	}, nil
}
