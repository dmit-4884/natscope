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
	if v.Operation == "subscription" {
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
// Violations that match no watched subject, and non-violation async errors,
// are retained for TakeRecent as the time-window fallback.
type PermissionWatcher struct {
	mu        sync.Mutex
	waiters   map[*permissionWaiter]struct{}
	observers map[*permissionObserver]struct{}
	lastErr   error
	lastAt    time.Time
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

// permissionWaiter is one Watch call: the subjects it listens for and the
// cancel that releases its in-flight request.
type permissionWaiter struct {
	subjects []string
	cancel   context.CancelFunc
	err      error
}

// matches reports whether the denied subject concerns this waiter: entries
// ending in "." match as prefixes (token boundary), anything else exactly.
func (w *permissionWaiter) matches(subject string) bool {
	for _, s := range w.subjects {
		if strings.HasSuffix(s, ".") {
			if strings.HasPrefix(subject, s) {
				return true
			}
		} else if subject == s {
			return true
		}
	}
	return false
}

// HandleAsyncError feeds an async error from nats.ErrorHandler. A permissions
// violation matching a watched subject cancels every matching in-flight call;
// everything else is retained for TakeRecent.
func (pw *PermissionWatcher) HandleAsyncError(err error) {
	if err == nil {
		return
	}

	var notify []func(error)
	var permErr error

	pw.mu.Lock()
	if v, ok := ParsePermissionViolation(err); ok {
		delivered := false
		for waiter := range pw.waiters {
			if waiter.err == nil && waiter.matches(v.Subject) {
				waiter.err = err
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

// Watch runs fn under a context that is canceled as soon as a permissions
// violation for one of the subjects arrives; the violation replaces fn's
// error. Subject entries ending in "." match as prefixes, others exactly.
func (pw *PermissionWatcher) Watch(ctx context.Context, subjects []string, fn func(ctx context.Context) error) error {
	ctx, end := pw.Begin(ctx, subjects)
	err := fn(ctx)
	if violation := end(); violation != nil {
		return violation
	}
	return err
}

// Begin watches subjects for an operation that outlives one call, such as a lister: the returned context is
// canceled as soon as a matching violation arrives. end stops the watch and returns that violation, if any.
func (pw *PermissionWatcher) Begin(ctx context.Context, subjects []string) (context.Context, func() error) {
	ctx, cancel := context.WithCancel(ctx)
	waiter := &permissionWaiter{subjects: subjects, cancel: cancel}
	pw.mu.Lock()
	pw.waiters[waiter] = struct{}{}
	pw.mu.Unlock()

	return ctx, func() error {
		pw.mu.Lock()
		delete(pw.waiters, waiter)
		violation := waiter.err
		pw.mu.Unlock()
		cancel()
		return violation
	}
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
