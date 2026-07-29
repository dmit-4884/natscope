// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"
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

// PermissionWatcher correlates out-of-band NATS permissions violations with
// in-flight requests. The server reports a violation asynchronously and never
// answers the offending request, so an uncorrelated caller blocks until its
// deadline; a caller under Watch is canceled the moment the violation arrives.
//
// Violations that match no watched subject, and non-violation async errors,
// are retained for TakeRecent as the time-window fallback.
type PermissionWatcher struct {
	mu      sync.Mutex
	waiters map[*permissionWaiter]struct{}
	lastErr error
	lastAt  time.Time
}

// NewPermissionWatcher returns a watcher ready to receive async errors.
func NewPermissionWatcher() *PermissionWatcher {
	return &PermissionWatcher{waiters: make(map[*permissionWaiter]struct{})}
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

	pw.mu.Lock()
	defer pw.mu.Unlock()

	if v, ok := ParsePermissionViolation(err); ok {
		delivered := false
		for waiter := range pw.waiters {
			if waiter.err == nil && waiter.matches(v.Subject) {
				waiter.err = err
				waiter.cancel()
				delivered = true
			}
		}
		if delivered {
			return
		}
	}

	pw.lastErr = err
	pw.lastAt = time.Now()
}

// Watch runs fn under a context that is canceled as soon as a permissions
// violation for one of the subjects arrives; the violation replaces fn's
// error. Subject entries ending in "." match as prefixes, others exactly.
func (pw *PermissionWatcher) Watch(ctx context.Context, subjects []string, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	waiter := &permissionWaiter{subjects: subjects, cancel: cancel}
	pw.mu.Lock()
	pw.waiters[waiter] = struct{}{}
	pw.mu.Unlock()

	err := fn(ctx)

	pw.mu.Lock()
	delete(pw.waiters, waiter)
	violation := waiter.err
	pw.mu.Unlock()

	if violation != nil {
		return violation
	}
	return err
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
