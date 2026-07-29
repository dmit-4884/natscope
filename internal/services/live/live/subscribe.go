// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"context"
	"log/slog"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"
)

// Subscribe runs one live subscription session; emit is invoked for every
// event (Batch/Stats/Error/ProtoReload) until ctx ends or emit errors.
func (s *Service) Subscribe(
	ctx context.Context,
	in *entities.LiveSubscribeRequest,
	emit func(*entities.LiveEvent) error,
) error {
	sess := &sessionState{}
	s.registerSession(sess)
	defer s.unregisterSession(sess)

	mode, maxDisplayRate, payloadCap := s.resolveSettings(ctx)
	// Request-level override has highest precedence over the user setting.
	if in.MaxPayloadBytes != nil {
		payloadCap = *in.MaxPayloadBytes
	}

	msgChan := make(chan *entities.NatsMessage, messageBufferSize)

	subscriptions, setupErr := s.startSubscriptions(ctx, in, mode, msgChan, sess)
	defer func() {
		for _, sub := range subscriptions {
			if sub != nil {
				_ = sub.Unsubscribe() //nolint:errcheck // best-effort teardown; peer disconnect is expected
			}
		}
	}()
	if setupErr != nil {
		return setupErr
	}

	return s.runLoop(ctx, sess, msgChan, maxDisplayRate, payloadCap, emit)
}

// resolveSettings reads SubscriptionMode, MaxDisplayRate, and payload cap
// from settings; unreadable settings fall through to server defaults.
func (s *Service) resolveSettings(ctx context.Context) (mode string, maxDisplayRate, payloadCap int32) {
	mode = subscriptionModeCoreNATS
	payloadCap = entities.DefaultMaxPayloadBytesInList
	cfg, err := s.settingsService.Get(ctx)
	if err != nil || cfg == nil {
		return mode, 0, payloadCap
	}
	if cfg.Live != nil {
		if cfg.Live.SubscriptionMode != nil {
			mode = *cfg.Live.SubscriptionMode
		}
		if cfg.Live.MaxDisplayRate != nil {
			maxDisplayRate = *cfg.Live.MaxDisplayRate
		}
	}
	if cfg.Messages != nil && cfg.Messages.MaxPayloadBytesInList != nil {
		payloadCap = *cfg.Messages.MaxPayloadBytesInList
	}
	return mode, maxDisplayRate, payloadCap
}

// startSubscriptions resolves every target into concrete NATS subscriptions;
// per-target failures are tolerated as long as at least one succeeds.
func (s *Service) startSubscriptions(
	ctx context.Context,
	in *entities.LiveSubscribeRequest,
	mode string,
	msgChan chan<- *entities.NatsMessage,
	sess *sessionState,
) ([]entities.Subscription, error) {
	handler := func(msg *entities.NatsMessage) {
		if natsutil.IsInternalSubject(msg.Subject) {
			return
		}
		select {
		case msgChan <- msg:
		default:
			// Buffer full — drop the message and count it in LiveStats.MessagesDropped.
			sess.messagesDropped.Add(1)
		}
	}

	var subs []entities.Subscription
	var lastErr error
	for _, target := range in.Subscriptions {
		if target == nil || target.Subject == "" {
			continue
		}
		targetSubs, err := s.subscribeTarget(ctx, in.ConnectionId, target, mode, handler)
		if err != nil {
			lastErr = err
			continue
		}
		subs = append(subs, targetSubs...)
	}

	if len(subs) == 0 {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, errs.ErrLiveNoSubscriptions
	}
	return subs, nil
}

// subscribeTarget resolves a target into subscriptions: JetStream-ordered +
// StreamName yields one ordered sub, else one core-NATS sub per subject.
func (s *Service) subscribeTarget(
	ctx context.Context,
	connectionID string,
	target *entities.LiveSubscriptionTarget,
	mode string,
	handler entities.MessageHandler,
) ([]entities.Subscription, error) {
	streamName := ""
	if target.StreamName != nil {
		streamName = *target.StreamName
	}

	effectiveMode := s.resolveEffectiveMode(ctx, connectionID, streamName, mode)

	if effectiveMode == subscriptionModeJetStreamOrdered && streamName != "" {
		sub, err := s.natsService.SubscribeJetStream(
			ctx, connectionID, streamName, target.Subject, defaultDeliverPolicy, handler,
		)
		if err != nil {
			s.logger.Error("failed to subscribe via JetStream ordered consumer",
				slog.String("stream", streamName),
				slog.String("subject", target.Subject),
				slog.String("error", err.Error()))
			return nil, err
		}
		return []entities.Subscription{sub}, nil
	}

	subjects := []string{target.Subject}
	if streamName != "" {
		streamSubjects, err := s.natsService.GetStreamSubjects(ctx, connectionID, streamName)
		if err != nil {
			s.logger.Error("failed to get stream subjects",
				slog.String("stream", streamName),
				slog.String("error", err.Error()))
			return nil, err
		}
		subjects = streamSubjects
	}

	out := make([]entities.Subscription, 0, len(subjects))
	var lastErr error
	for _, subject := range subjects {
		sub, err := s.natsService.Subscribe(ctx, connectionID, subject, handler)
		if err != nil {
			s.logger.Error("failed to subscribe",
				slog.String("subject", subject),
				slog.String("error", err.Error()))
			lastErr = err
			continue
		}
		out = append(out, sub)
	}
	if len(out) == 0 {
		return nil, lastErr
	}
	return out, nil
}

// resolveEffectiveMode falls back to core NATS when a WorkQueue stream would
// be drained by a JS-Ordered (AckNone) consumer.
func (s *Service) resolveEffectiveMode(ctx context.Context, connectionID, streamName, mode string) string {
	if mode != subscriptionModeJetStreamOrdered || streamName == "" {
		return mode
	}
	info, err := s.natsService.GetStreamInfo(ctx, connectionID, streamName)
	if err != nil || info == nil {
		return mode
	}
	if info.Config.Retention == entities.RetentionWorkQueue {
		s.logger.Warn("downgrading live subscription to core_nats: WorkQueue stream would be drained by a JS-Ordered consumer",
			slog.String("stream", streamName))
		return subscriptionModeCoreNATS
	}
	return mode
}
