// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// subscriptionWrapper wraps nats.Subscription to implement
// entities.Subscription.
type subscriptionWrapper struct {
	sub *nats.Subscription
}

func (w *subscriptionWrapper) Unsubscribe() error {
	return w.sub.Unsubscribe()
}

// Subscribe creates a Core NATS subscription for the given subject.
func (c *Client) Subscribe(
	_ context.Context,
	subject string,
	handler entities.MessageHandler,
) (entities.Subscription, error) {
	natsHandler := func(msg *nats.Msg) {
		handler(&entities.NatsMessage{
			Subject: msg.Subject,
			Data:    msg.Data,
			Header:  maps.Clone(msg.Header),
		})
	}

	sub, err := c.conn.Subscribe(subject, natsHandler)
	if err != nil {
		return nil, wrapErr(err)
	}

	return &subscriptionWrapper{sub: sub}, nil
}

// jsSubscriptionWrapper wraps a jetstream consumer to implement
// entities.Subscription; Unsubscribe stops consuming and deletes the consumer.
// ephemeralCleanupTimeout bounds the best-effort deletion of an ephemeral
// consumer during subscription teardown.
const ephemeralCleanupTimeout = 5 * time.Second

type jsSubscriptionWrapper struct {
	consumeCtx   jetstream.ConsumeContext
	stream       jetstream.Stream
	consumerName string
}

func (s *jsSubscriptionWrapper) Unsubscribe() error {
	s.consumeCtx.Stop()
	if s.stream != nil && s.consumerName != "" {
		// Bound the best-effort cleanup so a wedged server can't block the
		// caller (live session teardown) indefinitely.
		ctx, cancel := corecontext.ApplyTimeout(context.Background(), ephemeralCleanupTimeout)
		defer cancel()
		_ = s.stream.DeleteConsumer(ctx, s.consumerName) //nolint:errcheck // best-effort cleanup
	}
	return nil
}

// SubscribeJetStream creates a JetStream ordered consumer subscription for live
// messages.
func (c *Client) SubscribeJetStream(
	ctx context.Context,
	streamName, subject, deliverPolicy string,
	handler entities.MessageHandler,
) (entities.Subscription, error) {
	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.Wrapf(err, "failed to get stream %q", streamName))
	}

	// Refuse AckNone on a WorkQueue stream — server would auto-ack on delivery and
	// drain the queue; server-side guard for direct gRPC callers.
	if info := stream.CachedInfo(); info != nil && info.Config.Retention == jetstream.WorkQueuePolicy {
		return nil, errs.ErrWorkQueueConsumerNotAllowed
	}

	var policy jetstream.DeliverPolicy
	switch deliverPolicy {
	case "all":
		policy = jetstream.DeliverAllPolicy
	case "last":
		policy = jetstream.DeliverLastPolicy
	case "last_per_subject":
		policy = jetstream.DeliverLastPerSubjectPolicy
	default:
		policy = jetstream.DeliverNewPolicy
	}

	consumerName := fmt.Sprintf("natscope-live-%s", nats.NewInbox()[7:])

	ephCfg := jetstream.ConsumerConfig{
		Name:              consumerName,
		FilterSubject:     subject,
		DeliverPolicy:     policy,
		AckPolicy:         jetstream.AckNonePolicy,
		InactiveThreshold: ephemeralConsumerInactiveThreshold,
	}

	consumer, err := stream.CreateConsumer(ctx, ephCfg)
	if err != nil {
		return nil, wrapErr(err)
	}

	consumeCtx, err := consumer.Consume(func(msg jetstream.Msg) {
		nm := &entities.NatsMessage{
			Subject: msg.Subject(),
			Data:    msg.Data(),
			Header:  maps.Clone(msg.Headers()),
			Stream:  streamName,
		}

		if md, mdErr := msg.Metadata(); mdErr == nil {
			seq := md.Sequence.Stream
			nm.Sequence = &seq
			nm.Timestamp = md.Timestamp
		}

		handler(nm)
	})
	if err != nil {
		_ = stream.DeleteConsumer(ctx, consumerName) //nolint:errcheck // best-effort cleanup
		return nil, wrapErr(errors.WrapOperation(err, "start consuming"))
	}

	return &jsSubscriptionWrapper{
		consumeCtx:   consumeCtx,
		stream:       stream,
		consumerName: consumerName,
	}, nil
}

// Health returns health status (RTT + state) for the connection.
func (c *Client) Health(_ context.Context) (*entities.ConnectionHealth, error) {
	health := &entities.ConnectionHealth{
		Id:  c.id,
		URL: c.url,
	}

	switch {
	case c.conn.IsConnected():
		health.IsConnected = true
		health.Status = "connected"
		health.ServerVersion = c.conn.ConnectedServerVersion()

		start := time.Now()
		err := c.conn.Flush()
		if err == nil {
			health.RTT = time.Since(start).String()
		} else {
			health.Status = "degraded"
			health.Error = err.Error()
		}
	case c.conn.IsReconnecting():
		health.IsReconnecting = true
		health.Status = "reconnecting"
	default:
		health.Status = "disconnected"
	}

	return health, nil
}

// GetStreamSubjects returns the subject patterns for a specific stream.
func (c *Client) GetStreamSubjects(ctx context.Context, streamName string) ([]string, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, c.defaultTimeout)
	defer cancel()

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get stream"))
	}

	info := stream.CachedInfo()
	if info == nil {
		return nil, wrapErr(errs.ErrStreamNotFound)
	}
	return info.Config.Subjects, nil
}
