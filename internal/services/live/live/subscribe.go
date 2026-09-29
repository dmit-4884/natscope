// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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
	if err := validateSubscriptionTargets(in.Subscriptions); err != nil {
		return err
	}
	targets := dedupeSubscriptionTargets(in.Subscriptions)

	sess := newSessionState(in.ConnectionId)
	s.registerSession(sess)
	defer s.unregisterSession(sess)

	mode, maxDisplayRate, payloadCap := s.resolveSettings(ctx)
	// A request cap overrides the user setting; 0 means omitted, not unlimited.
	if in.MaxPayloadBytes != nil && *in.MaxPayloadBytes > 0 {
		payloadCap = *in.MaxPayloadBytes
	}

	msgChan := make(chan *entities.NatsMessage, messageBufferSize)

	subscriptions, partialErrs, setupErr := s.startSubscriptions(ctx, in.ConnectionId, targets, mode, msgChan, sess)
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

	for _, pe := range partialErrs {
		if err := emit(&entities.LiveEvent{
			Error: &entities.LiveError{Code: "SUBSCRIBE_TARGET_FAILED", Message: pe.Error()},
		}); err != nil {
			return err
		}
	}

	return s.runLoop(ctx, sess, msgChan, maxDisplayRate, payloadCap, emit)
}

// validateSubscriptionTargets rejects a malformed subject pattern before any NATS work happens.
func validateSubscriptionTargets(targets []*entities.LiveSubscriptionTarget) error {
	for _, target := range targets {
		if target == nil {
			continue
		}
		if err := natsutil.ValidateSubjectPattern(target.Subject); err != nil {
			return err
		}
	}
	return nil
}

// dedupeSubscriptionTargets drops exact duplicates and literal subjects covered by a wildcard in the same stream grouping.
func dedupeSubscriptionTargets(targets []*entities.LiveSubscriptionTarget) []*entities.LiveSubscriptionTarget {
	type targetKey struct {
		subject    string
		streamName string
	}

	seen := make(map[targetKey]bool, len(targets))
	out := make([]*entities.LiveSubscriptionTarget, 0, len(targets))
	for _, t := range targets {
		if t == nil || t.Subject == "" {
			continue
		}
		key := targetKey{subject: t.Subject, streamName: streamNameOf(t)}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}

	covered := make([]bool, len(out))
	for i, a := range out {
		if isLiteralSubject(a.Subject) {
			continue // only a wildcard pattern can cover another target
		}
		for j, b := range out {
			if i == j || covered[j] || !isLiteralSubject(b.Subject) {
				continue
			}
			if streamNameOf(a) != streamNameOf(b) {
				continue
			}
			if natsutil.MatchSubject(a.Subject, b.Subject) {
				covered[j] = true
			}
		}
	}

	result := make([]*entities.LiveSubscriptionTarget, 0, len(out))
	for i, t := range out {
		if !covered[i] {
			result = append(result, t)
		}
	}
	return result
}

func streamNameOf(t *entities.LiveSubscriptionTarget) string {
	if t == nil || t.StreamName == nil {
		return ""
	}
	return *t.StreamName
}

// isLiteralSubject reports whether a validated subject contains no wildcard tokens.
func isLiteralSubject(subject string) bool {
	return !strings.ContainsAny(subject, "*>")
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
// it returns per-target failures as long as at least one target succeeds.
func (s *Service) startSubscriptions(
	ctx context.Context,
	connectionID string,
	targets []*entities.LiveSubscriptionTarget,
	mode string,
	msgChan chan<- *entities.NatsMessage,
	sess *sessionState,
) ([]entities.Subscription, []error, error) {
	var subs []entities.Subscription
	var partialErrs []error
	for _, target := range targets {
		handler := s.buildMessageHandler(target.Subject, msgChan, sess)
		targetSubs, err := s.subscribeTarget(ctx, connectionID, target, mode, handler)
		if err != nil {
			partialErrs = append(partialErrs, fmt.Errorf("subject %q: %w", target.Subject, err))
			continue
		}
		subs = append(subs, targetSubs...)
	}

	if len(subs) == 0 {
		if len(partialErrs) > 0 {
			return nil, nil, partialErrs[len(partialErrs)-1]
		}
		return nil, nil, errs.ErrLiveNoSubscriptions
	}
	return subs, partialErrs, nil
}

// buildMessageHandler builds the per-target delivery callback. Internal subjects pass only when the
// target's own pattern names an internal namespace.
func (s *Service) buildMessageHandler(
	targetSubject string,
	msgChan chan<- *entities.NatsMessage,
	sess *sessionState,
) entities.MessageHandler {
	allowInternal := natsutil.IsInternalSubject(targetSubject)
	return func(msg *entities.NatsMessage) {
		if !allowInternal && natsutil.IsInternalSubject(msg.Subject) {
			return
		}
		sess.totalMessages.Add(1)
		size := int64(len(msg.Data))
		if sess.bufferedBytes.Add(size) > maxBufferedBytes {
			sess.bufferedBytes.Add(-size)
			sess.messagesDropped.Add(1)
			return
		}
		select {
		case msgChan <- msg:
		default:
			sess.bufferedBytes.Add(-size)
			sess.messagesDropped.Add(1)
		}
	}
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
	streamName := streamNameOf(target)

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
