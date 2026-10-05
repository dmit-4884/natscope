// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

type fakeSubscription struct{}

func (fakeSubscription) Unsubscribe() error { return nil }

type fakeSubscriber struct {
	natssvc.Subscriber

	mu       sync.Mutex
	onDenied map[string]func(error)
	handlers map[string]entities.MessageHandler
}

func (f *fakeSubscriber) Subscribe(
	_ context.Context,
	_ string,
	subject string,
	handler entities.MessageHandler,
	onDenied func(error),
) (entities.Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onDenied[subject] = onDenied
	f.handlers[subject] = handler
	return fakeSubscription{}, nil
}

func (f *fakeSubscriber) OnDisconnect(func(string)) {}

func (f *fakeSubscriber) deny(t *testing.T, subject string) {
	t.Helper()
	f.mu.Lock()
	fn := f.onDenied[subject]
	f.mu.Unlock()
	require.NotNil(t, fn, "no onDenied registered for %q", subject)
	fn(&errs.NATSPermissionError{Operation: errs.PermissionOperationSubscribe, Subject: subject})
}

func (f *fakeSubscriber) deliver(t *testing.T, subject string, msg *entities.NatsMessage) {
	t.Helper()
	f.mu.Lock()
	handler := f.handlers[subject]
	f.mu.Unlock()
	require.NotNil(t, handler)
	handler(msg)
}

type fakeDecoder struct{}

func (fakeDecoder) Reset()                                                        {}
func (fakeDecoder) Init(context.Context)                                          {}
func (fakeDecoder) Ready() bool                                                   { return true }
func (fakeDecoder) Decode(context.Context, []byte, string) *entities.DecodeResult { return nil }

type fakeCodec struct{ protosvc.Codec }

func (fakeCodec) NewLiveDecoder(bool) protosvc.LiveDecoder { return fakeDecoder{} }

type fakeSettings struct{ settingssvc.Service }

func (fakeSettings) Get(context.Context) (*entities.UserSettings, error) {
	return nil, errors.New("no settings")
}

func TestSubscribe_ReportsADeniedSubjectAndKeepsTheOthers(t *testing.T) {
	t.Parallel()

	sub := &fakeSubscriber{onDenied: map[string]func(error){}, handlers: map[string]entities.MessageHandler{}}
	svc := New(nil, sub, fakeCodec{}, fakeSettings{})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	events := make(chan *entities.LiveEvent, 16)
	done := make(chan error, 1)
	go func() {
		done <- svc.Subscribe(ctx, &entities.LiveSubscribeRequest{
			ConnectionId: "conn",
			Subscriptions: []*entities.LiveSubscriptionTarget{
				{Subject: "orders.>"},
				{Subject: "secret.>"},
			},
		}, func(ev *entities.LiveEvent) error {
			events <- ev
			return nil
		})
	}()

	require.Eventually(t, func() bool {
		sub.mu.Lock()
		defer sub.mu.Unlock()
		return len(sub.onDenied) == 2
	}, 5*time.Second, 10*time.Millisecond)

	sub.deny(t, "secret.>")
	sub.deny(t, "secret.>")
	denied := nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Error != nil })
	assert.Equal(t, liveErrSubscribePermissionDenied, denied.Error.Code)
	assert.Equal(t, `no permission to subscribe to "secret.>"`, denied.Error.Message)
	assert.Equal(t, &entities.AccessCheck{
		Status:    entities.AccessDenied,
		Operation: errs.PermissionOperationSubscribe,
		Subject:   "secret.>",
	}, denied.Error.Access)

	sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte(`{}`), Reply: "_INBOX.x"})
	batch := nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Batch != nil })
	require.Len(t, batch.Batch.Messages, 1)
	assert.Equal(t, "_INBOX.x", batch.Batch.Messages[0].NatsMessage.Reply)

	select {
	case ev := <-events:
		assert.Nil(t, ev.Error, "the same subject is reported once")
	default:
	}

	cancel()
	require.NoError(t, <-done)
}

func TestSubscribe_ACoveredSubjectTakesOverWhenItsWildcardIsDenied(t *testing.T) {
	t.Parallel()

	sub := &fakeSubscriber{onDenied: map[string]func(error){}, handlers: map[string]entities.MessageHandler{}}
	svc := New(nil, sub, fakeCodec{}, fakeSettings{})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	events := make(chan *entities.LiveEvent, 16)
	done := make(chan error, 1)
	go func() {
		done <- svc.Subscribe(ctx, &entities.LiveSubscribeRequest{
			ConnectionId: "conn",
			Subscriptions: []*entities.LiveSubscriptionTarget{
				{Subject: "orders.>"},
				{Subject: "orders.created"},
			},
		}, func(ev *entities.LiveEvent) error {
			events <- ev
			return nil
		})
	}()

	require.Eventually(t, func() bool {
		sub.mu.Lock()
		defer sub.mu.Unlock()
		return len(sub.handlers) == 2
	}, 5*time.Second, 10*time.Millisecond, "a literal subject covered by a wildcard is subscribed too")

	sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte("early")})
	sub.deliver(t, "orders.created", &entities.NatsMessage{Subject: "orders.created", Data: []byte("early")})
	sub.deny(t, "orders.>")
	nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Error != nil })
	sub.deliver(t, "orders.created", &entities.NatsMessage{Subject: "orders.created", Data: []byte("late")})

	assert.Equal(t, map[string]int{"early": 1, "late": 1}, collectPayloads(t, events, "late"),
		"while the wildcard delivers, the covered copy is dropped; once it is denied, the covered subject takes over")

	cancel()
	require.NoError(t, <-done)
}

func TestSubscribe_OverlappingSubjectsDeliverAMessageOnce(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		targets  []string
		subject  string
		carriers []string
	}{
		{name: "wildcard under a wider wildcard", targets: []string{">", "orders.>"}, subject: "orders.created", carriers: []string{">", "orders.>"}},
		{name: "wider wildcard listed second", targets: []string{"orders.>", ">"}, subject: "orders.created", carriers: []string{"orders.>", ">"}},
		{name: "partial overlap", targets: []string{"orders.*", "*.created"}, subject: "orders.created", carriers: []string{"orders.*", "*.created"}},
		{name: "literal under a wildcard", targets: []string{"orders.created", "orders.>"}, subject: "orders.created", carriers: []string{"orders.created", "orders.>"}},
		{name: "system subject under the full wildcard", targets: []string{">", "$JS.EVENT.>"}, subject: "$JS.EVENT.ADVISORY.X", carriers: []string{">", "$JS.EVENT.>"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sub, events, stop := startFakeSession(t, tt.targets...)
			defer stop()

			for _, carrier := range tt.carriers {
				sub.deliver(t, carrier, &entities.NatsMessage{Subject: tt.subject, Data: []byte("once")})
			}
			sub.deliver(t, tt.carriers[0], &entities.NatsMessage{Subject: tt.subject, Data: []byte("end")})
			for _, carrier := range tt.carriers[1:] {
				sub.deliver(t, carrier, &entities.NatsMessage{Subject: tt.subject, Data: []byte("end")})
			}

			assert.Equal(t, map[string]int{"once": 1, "end": 1}, collectPayloads(t, events, "end"))
		})
	}
}

func TestSubscribe_TheNextSubjectDeliversWhenAnOverlappingOneIsDenied(t *testing.T) {
	t.Parallel()

	sub, events, stop := startFakeSession(t, ">", "orders.>")
	defer stop()

	sub.deny(t, ">")
	nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Error != nil })
	sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte("end")})

	assert.Equal(t, map[string]int{"end": 1}, collectPayloads(t, events, "end"))
}

func TestSubscribe_ARequestedDisplayRateOverridesTheSetting(t *testing.T) {
	t.Parallel()

	rate := int32(1)
	sub, events, stop := startFakeSessionWith(t, &entities.LiveSubscribeRequest{
		ConnectionId:   "conn",
		Subscriptions:  []*entities.LiveSubscriptionTarget{{Subject: "orders.>"}},
		MaxDisplayRate: &rate,
	})
	defer stop()

	for range 3 {
		sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte("burst")})
	}
	sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte("end")})

	seen := map[string]int{}
	deadline := time.After(700 * time.Millisecond)
	for collecting := true; collecting; {
		select {
		case ev := <-events:
			if ev.Batch == nil {
				continue
			}
			for _, m := range ev.Batch.Messages {
				seen[string(m.NatsMessage.Data)]++
			}
		case <-deadline:
			collecting = false
		}
	}
	assert.Equal(t, map[string]int{"burst": 1}, seen, "a rate of 1 msg/s shows the first message of a burst and skips the rest")
}

func startFakeSession(t *testing.T, subjects ...string) (*fakeSubscriber, <-chan *entities.LiveEvent, func()) {
	t.Helper()
	targets := make([]*entities.LiveSubscriptionTarget, 0, len(subjects))
	for _, subject := range subjects {
		targets = append(targets, &entities.LiveSubscriptionTarget{Subject: subject})
	}
	return startFakeSessionWith(t, &entities.LiveSubscribeRequest{ConnectionId: "conn", Subscriptions: targets})
}

func startFakeSessionWith(t *testing.T, req *entities.LiveSubscribeRequest) (*fakeSubscriber, <-chan *entities.LiveEvent, func()) {
	t.Helper()

	sub := &fakeSubscriber{onDenied: map[string]func(error){}, handlers: map[string]entities.MessageHandler{}}
	svc := New(nil, sub, fakeCodec{}, fakeSettings{})
	ctx, cancel := context.WithCancel(t.Context())

	events := make(chan *entities.LiveEvent, 64)
	done := make(chan error, 1)
	go func() {
		done <- svc.Subscribe(ctx, req, func(ev *entities.LiveEvent) error {
			events <- ev
			return nil
		})
	}()

	require.Eventually(t, func() bool {
		sub.mu.Lock()
		defer sub.mu.Unlock()
		return len(sub.handlers) == len(req.Subscriptions)
	}, 5*time.Second, 10*time.Millisecond)

	return sub, events, func() {
		cancel()
		require.NoError(t, <-done)
	}
}

func collectPayloads(t *testing.T, events <-chan *entities.LiveEvent, last string) map[string]int {
	t.Helper()
	seen := map[string]int{}
	for seen[last] == 0 {
		batch := nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Batch != nil })
		for _, m := range batch.Batch.Messages {
			seen[string(m.NatsMessage.Data)]++
		}
	}
	select {
	case ev := <-events:
		if ev.Batch != nil {
			for _, m := range ev.Batch.Messages {
				seen[string(m.NatsMessage.Data)]++
			}
		}
	case <-time.After(300 * time.Millisecond):
	}
	return seen
}

func nextEvent(t *testing.T, events <-chan *entities.LiveEvent, match func(*entities.LiveEvent) bool) *entities.LiveEvent {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case ev := <-events:
			if match(ev) {
				return ev
			}
		case <-deadline:
			t.Fatal("no matching live event")
			return nil
		}
	}
}

func TestSubscribe_ExcludedSubjectsDoNotUseTheDisplayRate(t *testing.T) {
	t.Parallel()

	rate := int32(1)
	sub, events, stop := startFakeSessionWith(t, &entities.LiveSubscribeRequest{
		ConnectionId:    "conn",
		Subscriptions:   []*entities.LiveSubscriptionTarget{{Subject: ">"}},
		MaxDisplayRate:  &rate,
		ExcludeSubjects: []string{"metrics.>"},
	})
	defer stop()

	for range 5 {
		sub.deliver(t, ">", &entities.NatsMessage{Subject: "metrics.cpu", Data: []byte("noise")})
	}
	sub.deliver(t, ">", &entities.NatsMessage{Subject: "orders.created", Data: []byte("wanted")})

	assert.Equal(t, map[string]int{"wanted": 1}, collectPayloads(t, events, "wanted"))
	var stats *entities.LiveStats
	deadline := time.After(2*statsInterval + 2*time.Second)
	for stats == nil {
		select {
		case ev := <-events:
			stats = ev.Stats
		case <-deadline:
			t.Fatal("no stats")
		}
	}
	assert.Zero(t, stats.MessagesDropped, "a muted message is not a skipped one")
	assert.Equal(t, int64(5), stats.SubjectCounts["metrics.cpu"], "muted subjects still count")
}

func TestSubscribe_AnAllowedSubjectKeepsItsMessagesUntilAnEarlierWildcardProvesLive(t *testing.T) {
	t.Parallel()

	sub, events, stop := startFakeSession(t, ">", "orders.>")
	defer stop()

	sub.deliver(t, "orders.>", &entities.NatsMessage{Subject: "orders.created", Data: []byte("first")})

	assert.Equal(t, map[string]int{"first": 1}, collectPayloads(t, events, "first"),
		"the wildcard may still be refused, so the covered subject delivers until the wildcard has delivered")
}
