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

	sub.deliver(t, "orders.created", &entities.NatsMessage{Subject: "orders.created", Data: []byte("early")})
	sub.deny(t, "orders.>")
	nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Error != nil })
	sub.deliver(t, "orders.created", &entities.NatsMessage{Subject: "orders.created", Data: []byte("late")})

	batch := nextEvent(t, events, func(ev *entities.LiveEvent) bool { return ev.Batch != nil })
	require.Len(t, batch.Batch.Messages, 1, "while the wildcard is live it delivers, so the covered copy is dropped")
	assert.Equal(t, "late", string(batch.Batch.Messages[0].NatsMessage.Data))

	cancel()
	require.NoError(t, <-done)
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
