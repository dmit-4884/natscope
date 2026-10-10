// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"maps"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// subscriptionWrapper wraps nats.Subscription to implement
// entities.Subscription.
type subscriptionWrapper struct {
	sub           *nats.Subscription
	stopObserving func()
	subject       string
	hiddenPrefix  string
}

func (w *subscriptionWrapper) Unsubscribe() error {
	if w.stopObserving != nil {
		w.stopObserving()
	}
	return w.sub.Unsubscribe()
}

func (w *subscriptionWrapper) Delivers(subject string) bool {
	if w.hiddenPrefix != "" && strings.HasPrefix(subject, w.hiddenPrefix) {
		return false
	}
	return natsutil.MatchSubject(w.subject, subject)
}

// Subscribe creates a Core NATS subscription for the given subject. Replies to this connection's own requests,
// under its inbox prefix, reach only a subject that names that prefix.
func (c *Client) Subscribe(
	_ context.Context,
	subject string,
	handler entities.MessageHandler,
	onDenied func(error),
) (entities.Subscription, error) {
	if err := validateNATSSubjectLength("subject", subject); err != nil {
		return nil, wrapErr(err)
	}

	var hiddenPrefix string
	if ownInbox := c.inboxPrefix(); !strings.HasPrefix(subject, ownInbox) {
		hiddenPrefix = ownInbox
	}
	natsHandler := func(msg *nats.Msg) {
		if hiddenPrefix != "" && strings.HasPrefix(msg.Subject, hiddenPrefix) {
			return
		}
		// Stamp the receive time at delivery, not at the later batch conversion.
		handler(&entities.NatsMessage{
			Subject:   msg.Subject,
			Data:      msg.Data,
			Header:    maps.Clone(msg.Header),
			Reply:     msg.Reply,
			Timestamp: time.Now(),
		})
	}

	var stop func()
	if onDenied != nil {
		stop = c.permWatch.Observe(PermissionViolation{Operation: violatedSubscription, Subject: subject}, onDenied)
	}
	sub, err := c.conn.Subscribe(subject, natsHandler)
	if err != nil {
		if stop != nil {
			stop()
		}
		return nil, wrapErr(err)
	}

	return &subscriptionWrapper{sub: sub, stopObserving: stop, subject: subject, hiddenPrefix: hiddenPrefix}, nil
}

// jsSubscriptionWrapper wraps a jetstream consumer to implement
// entities.Subscription; Unsubscribe stops consuming and deletes the consumer.
// ephemeralCleanupTimeout bounds the best-effort deletion of an ephemeral
// consumer during subscription teardown.
const ephemeralCleanupTimeout = 5 * time.Second

type jsSubscriptionWrapper struct {
	consumeCtx jetstream.ConsumeContext
	stream     jetstream.Stream
	consumer   jetstream.Consumer
	forget     func()
	filter     string
}

func (s *jsSubscriptionWrapper) Delivers(subject string) bool {
	return s.filter == "" || natsutil.MatchSubject(s.filter, subject)
}

func (s *jsSubscriptionWrapper) Unsubscribe() error {
	s.consumeCtx.Stop()
	if info := s.consumer.CachedInfo(); s.stream != nil && info != nil {
		// Bound the best-effort cleanup so a wedged server can't block the
		// caller (live session teardown) indefinitely.
		ctx, cancel := corecontext.ApplyTimeout(context.Background(), ephemeralCleanupTimeout)
		defer cancel()
		_ = s.stream.DeleteConsumer(ctx, info.Name) //nolint:errcheck // best-effort cleanup
	}
	if s.forget != nil {
		s.forget()
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
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSSubjectLength("subject", subject); err != nil {
		return nil, wrapErr(err)
	}

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

	namePrefix := liveConsumerPrefix + nats.NewInbox()[7:]
	forget := c.trackOwnConsumerSeries(namePrefix)
	consumer, err := stream.OrderedConsumer(ctx, jetstream.OrderedConsumerConfig{
		FilterSubjects:    []string{subject},
		DeliverPolicy:     policy,
		InactiveThreshold: ephemeralConsumerInactiveThreshold,
		NamePrefix:        namePrefix,
	})
	if err != nil {
		forget()
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
		if info := consumer.CachedInfo(); info != nil {
			_ = stream.DeleteConsumer(ctx, info.Name) //nolint:errcheck // best-effort cleanup
		}
		forget()
		return nil, wrapErr(errors.WrapOperation(err, "start consuming"))
	}

	return &jsSubscriptionWrapper{
		consumeCtx: consumeCtx,
		stream:     stream,
		consumer:   consumer,
		forget:     forget,
		filter:     subject,
	}, nil
}

// Health returns health status (RTT + state) for the connection.
func (c *Client) Health(_ context.Context) (*entities.ConnectionHealth, error) {
	health := &entities.ConnectionHealth{
		Id:  c.id,
		URL: c.url,
	}

	switch {
	case c.IsConnected():
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
	case c.IsReconnecting():
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
