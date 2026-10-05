// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/dmit-4884/natscope/internal/errs"
)

// permissionViolationRe extracts the denied operation and subject from the
// text of a NATS permissions-violation async error; the server reports these
// out-of-band with no SDK sentinel attached.
var permissionViolationRe = regexp.MustCompile(`(?i)permissions violation for (publish|subscription) to "?([^\s"]+)"?`)

// The operations a permissions violation names.
const (
	violatedPublish      = "publish"
	violatedSubscription = "subscription"
)

// PermissionViolation is a parsed NATS permissions-violation async error.
type PermissionViolation struct {
	// Operation is the denied verb, "publish" or "subscription".
	Operation string
	// Subject is the subject the server denied.
	Subject string
}

// ParsePermissionViolation extracts the denied operation and subject from a
// NATS async permissions-violation error; ok is false for any other error.
func ParsePermissionViolation(err error) (PermissionViolation, bool) {
	if err == nil {
		return PermissionViolation{}, false
	}
	m := permissionViolationRe.FindStringSubmatch(err.Error())
	if m == nil {
		return PermissionViolation{}, false
	}
	return PermissionViolation{Operation: strings.ToLower(m[1]), Subject: m[2]}, true
}

func (v PermissionViolation) asError(cause error) *errs.NATSPermissionError {
	operation := errs.PermissionOperationPublish
	if v.Operation == violatedSubscription {
		operation = errs.PermissionOperationSubscribe
	}
	return &errs.NATSPermissionError{Operation: operation, Subject: v.Subject, Cause: cause}
}

// widenInboxDenial reports a refused reply inbox as the whole inbox namespace, the permission the user lacks.
func widenInboxDenial(err error, inboxPrefix string) error {
	permErr, ok := errors.AsType[*errs.NATSPermissionError](err)
	if !ok || permErr.Operation != errs.PermissionOperationSubscribe || !strings.HasPrefix(permErr.Subject, inboxPrefix) {
		return err
	}
	return &errs.NATSPermissionError{Operation: permErr.Operation, Subject: inboxPrefix + ">", Cause: permErr.Cause}
}

// PermissionWatcher correlates out-of-band NATS permissions violations with
// in-flight requests. The server reports a violation asynchronously and never
// answers the offending request, so an uncorrelated caller blocks until its
// deadline; a caller under Watch is canceled the moment the violation arrives.
//
// A publish violation concerns the calls that publish to its subject; a
// subscription violation concerns only the calls that wait on the refused
// subscription. The server refuses the connection's shared reply subscription
// once, when it is made, so that refusal is kept until a reconnect and fails
// every later request at once.
//
// Violations that match no watched call, and non-violation async errors,
// are retained for TakeRecent as the time-window fallback.
type PermissionWatcher struct {
	mu        sync.Mutex
	waiters   map[*permissionWaiter]struct{}
	observers map[*permissionObserver]struct{}
	lastErr   error
	lastAt    time.Time

	inboxPrefix string
	replies     string
	repliesErr  error
}

// NewPermissionWatcher returns a watcher ready to receive async errors.
func NewPermissionWatcher() *PermissionWatcher {
	return &PermissionWatcher{
		waiters:   make(map[*permissionWaiter]struct{}),
		observers: make(map[*permissionObserver]struct{}),
	}
}

type permissionObserver struct {
	match PermissionViolation
	fn    func(error)
}

// Observe calls fn with *errs.NATSPermissionError for every violation equal to match until stop is called.
func (pw *PermissionWatcher) Observe(match PermissionViolation, fn func(error)) (stop func()) {
	observer := &permissionObserver{match: match, fn: fn}
	pw.mu.Lock()
	pw.observers[observer] = struct{}{}
	pw.mu.Unlock()
	return func() {
		pw.mu.Lock()
		delete(pw.observers, observer)
		pw.mu.Unlock()
	}
}

// TrackReplies names the connection's shared reply subscription, which every request waits on, and the prefix of
// its inboxes, whose whole namespace a refusal reports as the missing permission.
func (pw *PermissionWatcher) TrackReplies(inboxPrefix, subject string) {
	pw.mu.Lock()
	pw.inboxPrefix, pw.replies = inboxPrefix, subject
	pw.mu.Unlock()
}

// ResetReplies forgets a refused reply subscription; a reconnect makes it again and the server decides anew.
func (pw *PermissionWatcher) ResetReplies() {
	pw.mu.Lock()
	pw.repliesErr = nil
	pw.mu.Unlock()
}

// permissionWaiter is one watched call: the subjects it publishes to, the
// subscription it waits on, and the cancel that releases it.
type permissionWaiter struct {
	subjects []string
	replies  bool
	inbox    string
	cancel   context.CancelFunc
	err      error
}

// matches reports whether a violation concerns this waiter. Publish subjects
// ending in "." match as prefixes (token boundary), others exactly; a refused
// subscription matches only the shared reply subscription or the waiter's own inbox.
func (w *permissionWaiter) matches(v PermissionViolation, replies string) bool {
	if v.Operation != violatedPublish {
		return (w.replies && replies != "" && v.Subject == replies) || (w.inbox != "" && v.Subject == w.inbox)
	}
	for _, s := range w.subjects {
		if strings.HasSuffix(s, ".") {
			if strings.HasPrefix(v.Subject, s) {
				return true
			}
		} else if v.Subject == s {
			return true
		}
	}
	return false
}

// HandleAsyncError feeds an async error from nats.ErrorHandler. A permissions
// violation cancels every in-flight call it concerns; everything else is
// retained for TakeRecent.
func (pw *PermissionWatcher) HandleAsyncError(err error) {
	if err == nil {
		return
	}

	var notify []func(error)
	var permErr error

	pw.mu.Lock()
	if v, ok := ParsePermissionViolation(err); ok {
		delivered := false
		waiterErr := err
		if v.Operation != violatedPublish && pw.replies != "" && v.Subject == pw.replies {
			waiterErr = &errs.NATSPermissionError{Operation: errs.PermissionOperationSubscribe, Subject: pw.inboxPrefix + ">", Cause: err}
			pw.repliesErr = waiterErr
			delivered = true
		}
		for waiter := range pw.waiters {
			if waiter.err == nil && waiter.matches(v, pw.replies) {
				waiter.err = waiterErr
				waiter.cancel()
				delivered = true
			}
		}
		for observer := range pw.observers {
			if observer.match == v {
				notify = append(notify, observer.fn)
			}
		}
		if len(notify) > 0 {
			permErr = v.asError(err)
		}
		delivered = delivered || len(notify) > 0
		if !delivered {
			pw.lastErr, pw.lastAt = err, time.Now()
		}
	} else {
		pw.lastErr, pw.lastAt = err, time.Now()
	}
	pw.mu.Unlock()

	for _, fn := range notify {
		fn(permErr)
	}
}

// Watch runs a publish under a context that is canceled as soon as a
// violation for one of the subjects arrives; the violation replaces fn's
// error. Subject entries ending in "." match as prefixes, others exactly.
func (pw *PermissionWatcher) Watch(ctx context.Context, subjects []string, fn func(ctx context.Context) error) error {
	return pw.watch(ctx, &permissionWaiter{subjects: subjects}, fn)
}

// WatchRequest is Watch for a request whose reply comes through the shared
// reply subscription: a refusal of that subscription fails it too, at once
// when the refusal is already known.
func (pw *PermissionWatcher) WatchRequest(ctx context.Context, subjects []string, fn func(ctx context.Context) error) error {
	return pw.watch(ctx, &permissionWaiter{subjects: subjects, replies: true}, fn)
}

// WatchInbox is Watch for a call that subscribes to its own reply inbox.
func (pw *PermissionWatcher) WatchInbox(ctx context.Context, subjects []string, inbox string, fn func(ctx context.Context) error) error {
	return pw.watch(ctx, &permissionWaiter{subjects: subjects, inbox: inbox}, fn)
}

func (pw *PermissionWatcher) watch(ctx context.Context, waiter *permissionWaiter, fn func(ctx context.Context) error) error {
	ctx, end, refused := pw.begin(ctx, waiter)
	if refused {
		return end()
	}
	err := fn(ctx)
	if violation := end(); violation != nil {
		return violation
	}
	return err
}

// Begin watches a request that outlives one call, such as a lister: the
// returned context is canceled as soon as a matching violation arrives, or
// right away when the shared reply subscription is already refused. end stops
// the watch and returns that violation, if any.
func (pw *PermissionWatcher) Begin(ctx context.Context, subjects []string) (context.Context, func() error) {
	ctx, end, _ := pw.begin(ctx, &permissionWaiter{subjects: subjects, replies: true})
	return ctx, end
}

// begin registers waiter; refused reports a request whose reply subscription is already refused, which end returns.
func (pw *PermissionWatcher) begin(ctx context.Context, waiter *permissionWaiter) (_ context.Context, end func() error, refused bool) {
	ctx, cancel := context.WithCancel(ctx)
	waiter.cancel = cancel
	pw.mu.Lock()
	if waiter.replies && pw.repliesErr != nil {
		refused = true
		waiter.err = pw.repliesErr
		cancel()
	} else {
		pw.waiters[waiter] = struct{}{}
	}
	pw.mu.Unlock()

	return ctx, func() error {
		pw.mu.Lock()
		delete(pw.waiters, waiter)
		violation := waiter.err
		pw.mu.Unlock()
		cancel()
		return violation
	}, refused
}

// TakeRecent returns and clears the last uncorrelated async error when it
// arrived within window; an older retained error is discarded.
func (pw *PermissionWatcher) TakeRecent(window time.Duration) error {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	if pw.lastErr == nil {
		return nil
	}
	err := pw.lastErr
	pw.lastErr = nil
	if time.Since(pw.lastAt) > window {
		return nil
	}
	return err
}
