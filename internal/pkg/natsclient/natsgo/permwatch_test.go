// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/errs"
)

func TestWidenInboxDenial(t *testing.T) {
	t.Parallel()

	inbox := &errs.NATSPermissionError{Operation: errs.PermissionOperationSubscribe, Subject: "_INBOX.k3J9x.Qp2"}
	got, ok := errors.AsType[*errs.NATSPermissionError](widenInboxDenial(inbox, "_INBOX."))
	require.True(t, ok)
	assert.Equal(t, "_INBOX.>", got.Subject, "the missing permission is the inbox namespace, not one random inbox")
	assert.Equal(t, errs.PermissionOperationSubscribe, got.Operation)

	publish := &errs.NATSPermissionError{Operation: errs.PermissionOperationPublish, Subject: "$SRV.INFO"}
	assert.Same(t, publish, widenInboxDenial(publish, "_INBOX."))

	other := errors.New("boom")
	assert.Same(t, other, widenInboxDenial(other, "_INBOX."))
}

func TestParsePermissionViolation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		want   PermissionViolation
		wantOK bool
	}{
		{
			name:   "PublishQuoted",
			err:    errors.New(`nats: Permissions Violation for Publish to "$JS.API.CONSUMER.CREATE.ORDERS.natsgo-browse-x"`),
			want:   PermissionViolation{Operation: "publish", Subject: "$JS.API.CONSUMER.CREATE.ORDERS.natsgo-browse-x"},
			wantOK: true,
		},
		{
			name:   "SubscriptionUnquoted",
			err:    errors.New("nats: Permissions Violation for Subscription to orders.created"),
			want:   PermissionViolation{Operation: "subscription", Subject: "orders.created"},
			wantOK: true,
		},
		{
			name:   "LowercaseText",
			err:    errors.New(`permissions violation for publish to "$JS.API.STREAM.CREATE.FOO"`),
			want:   PermissionViolation{Operation: "publish", Subject: "$JS.API.STREAM.CREATE.FOO"},
			wantOK: true,
		},
		{name: "NotViolation", err: errors.New("nats: timeout"), wantOK: false},
		{name: "Nil", err: nil, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParsePermissionViolation(tt.err)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPermissionWatcher_ObserveReportsMatchingSubscriptionViolations(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	var got []error
	stop := pw.Observe(PermissionViolation{Operation: "subscription", Subject: "secret.>"}, func(err error) {
		got = append(got, err)
	})

	pw.HandleAsyncError(errors.New(`nats: Permissions Violation for Publish to "secret.>"`))
	other := errors.New(`nats: Permissions Violation for Subscription to "other.>"`)
	pw.HandleAsyncError(other)
	assert.Empty(t, got)
	assert.ErrorIs(t, pw.TakeRecent(time.Minute), other, "an unobserved violation stays for the fallback")

	denied := errors.New(`nats: Permissions Violation for Subscription to "secret.>"`)
	pw.HandleAsyncError(denied)
	require.Len(t, got, 1)
	assert.ErrorIs(t, got[0], denied)
	assert.NoError(t, pw.TakeRecent(time.Minute), "an observed violation is not left for the fallback")

	stop()
	pw.HandleAsyncError(denied)
	assert.Len(t, got, 1, "a stopped observer hears nothing")
	assert.ErrorIs(t, pw.TakeRecent(time.Minute), denied)
}

func TestPermissionWatcher_WatchCancelsOnMatchingViolation(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	violation := errors.New(`nats: Permissions Violation for Publish to "$JS.API.CONSUMER.CREATE.ORDERS.abc"`)

	err := pw.Watch(t.Context(), []string{"$JS.API.CONSUMER.CREATE.ORDERS."}, func(ctx context.Context) error {
		go pw.HandleAsyncError(violation)
		<-ctx.Done()
		return ctx.Err()
	})

	require.ErrorIs(t, err, violation)
	// Delivered to the waiter, so nothing remains for the fallback.
	assert.NoError(t, pw.TakeRecent(time.Minute))
}

func TestPermissionWatcher_WatchPassesThroughFnError(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	fnErr := errors.New("boom")

	err := pw.Watch(t.Context(), []string{"$JS.API.STREAM.CREATE.FOO"}, func(context.Context) error {
		return fnErr
	})

	require.ErrorIs(t, err, fnErr)
}

func TestPermissionWatcher_UnmatchedViolationGoesToFallback(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	violation := errors.New(`nats: Permissions Violation for Publish to "$JS.API.STREAM.CREATE.OTHER"`)

	err := pw.Watch(t.Context(), []string{"$JS.API.CONSUMER.CREATE.ORDERS."}, func(context.Context) error {
		pw.HandleAsyncError(violation)
		return nil
	})

	require.NoError(t, err)
	assert.ErrorIs(t, pw.TakeRecent(time.Minute), violation)
	// Second take is empty.
	assert.NoError(t, pw.TakeRecent(time.Minute))
}

func TestPermissionWatcher_ExactMatchDoesNotCoverSiblingStream(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	violation := errors.New(`nats: Permissions Violation for Publish to "$JS.API.STREAM.CREATE.ORDERS_V2"`)

	err := pw.Watch(t.Context(), []string{"$JS.API.STREAM.CREATE.ORDERS"}, func(context.Context) error {
		pw.HandleAsyncError(violation)
		return nil
	})

	require.NoError(t, err)
	assert.ErrorIs(t, pw.TakeRecent(time.Minute), violation)
}

func TestPermissionWatcher_ViolationDeliveredToAllMatchingWaiters(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	violation := errors.New(`nats: Permissions Violation for Publish to "$JS.API.CONSUMER.CREATE.ORDERS.abc"`)

	first := make(chan error, 1)
	go func() {
		first <- pw.Watch(t.Context(), []string{"$JS.API.CONSUMER.CREATE.ORDERS."}, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
	}()

	err := pw.Watch(t.Context(), []string{"$JS.API.CONSUMER.CREATE.ORDERS."}, func(ctx context.Context) error {
		// Wait for the first waiter to register, then fire the violation.
		assert.Eventually(t, func() bool {
			pw.mu.Lock()
			defer pw.mu.Unlock()
			return len(pw.waiters) == 2
		}, time.Second, time.Millisecond)
		pw.HandleAsyncError(violation)
		<-ctx.Done()
		return ctx.Err()
	})

	require.ErrorIs(t, err, violation)
	require.ErrorIs(t, <-first, violation)
}

func TestPermissionWatcher_TakeRecentExpiresOutsideWindow(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	pw.HandleAsyncError(errors.New("nats: slow consumer"))

	assert.NoError(t, pw.TakeRecent(-time.Nanosecond))
	assert.NoError(t, pw.TakeRecent(time.Minute))
}

func TestPermissionWatcher_BeginWatchesUntilEnd(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	subjects := []string{"$JS.API.CONSUMER.LIST.SECRET"}
	violation := errors.New(`nats: Permissions Violation for Publish to "$JS.API.CONSUMER.LIST.SECRET"`)

	ctx, end := pw.Begin(t.Context(), subjects)
	pw.HandleAsyncError(violation)
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("the violation must cancel the watched context")
	}
	require.ErrorIs(t, end(), violation)

	_, end = pw.Begin(t.Context(), subjects)
	require.NoError(t, end())
	pw.HandleAsyncError(violation)
	assert.ErrorIs(t, pw.TakeRecent(time.Minute), violation, "after end the subject is no longer watched")
}

func TestPermissionWatcher_MatchesTheOperation(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	pw.TrackReplies("_INBOX.", "_INBOX.mux.*")
	refusedSubject := errors.New(`nats: Permissions Violation for Subscription to "orders.get"`)
	refusedNamespace := errors.New(`nats: Permissions Violation for Subscription to "_INBOX.>"`)

	err := pw.WatchRequest(t.Context(), []string{"orders.get"}, func(context.Context) error {
		pw.HandleAsyncError(refusedSubject)
		pw.HandleAsyncError(refusedNamespace)
		return nil
	})
	require.NoError(t, err, "another subscription's refusal is not this request's")

	err = pw.WatchInbox(t.Context(), []string{"$SRV.INFO"}, "_INBOX.mux.own", func(context.Context) error {
		pw.HandleAsyncError(refusedNamespace)
		return nil
	})
	require.NoError(t, err)

	own := errors.New(`nats: Permissions Violation for Subscription to "_INBOX.mux.own"`)
	err = pw.WatchInbox(t.Context(), []string{"$SRV.INFO"}, "_INBOX.mux.own", func(ctx context.Context) error {
		go pw.HandleAsyncError(own)
		<-ctx.Done()
		return ctx.Err()
	})
	require.ErrorIs(t, err, own)
}

func TestPermissionWatcher_RefusedRepliesStayRefusedUntilReconnect(t *testing.T) {
	t.Parallel()

	pw := NewPermissionWatcher()
	pw.TrackReplies("_INBOX.", "_INBOX.mux.*")
	refused := errors.New(`nats: Permissions Violation for Subscription to "_INBOX.mux.*"`)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	err := pw.WatchRequest(ctx, []string{"svc.echo"}, func(ctx context.Context) error {
		go pw.HandleAsyncError(refused)
		<-ctx.Done()
		return ctx.Err()
	})
	require.ErrorIs(t, err, refused)

	ran := false
	err = pw.WatchRequest(t.Context(), []string{"svc.echo"}, func(context.Context) error {
		ran = true
		return nil
	})
	require.ErrorIs(t, err, refused, "the server refuses the shared reply subscription only once")
	assert.False(t, ran)
	require.NoError(t, pw.Watch(t.Context(), []string{"svc.echo"}, func(context.Context) error { return nil }), "a plain publish needs no replies")

	pw.ResetReplies()
	require.NoError(t, pw.WatchRequest(t.Context(), []string{"svc.echo"}, func(context.Context) error { return nil }))
}
