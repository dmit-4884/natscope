// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import "sync"

// sourceLocks serializes compile->persist per source id so a slow compile of a
// stale disk snapshot can't overwrite a fresh one. Entries are refcounted and
// dropped once idle, so deleted sources don't accumulate.
type sourceLocks struct {
	mu sync.Mutex
	m  map[string]*sourceLock
}

type sourceLock struct {
	mu   sync.Mutex
	refs int
}

func (l *sourceLocks) lock(sourceID string) func() {
	l.mu.Lock()
	if l.m == nil {
		l.m = make(map[string]*sourceLock)
	}
	sl, ok := l.m[sourceID]
	if !ok {
		sl = &sourceLock{}
		l.m[sourceID] = sl
	}
	sl.refs++
	l.mu.Unlock()

	sl.mu.Lock()
	return func() {
		sl.mu.Unlock()
		l.mu.Lock()
		sl.refs--
		if sl.refs == 0 {
			delete(l.m, sourceID)
		}
		l.mu.Unlock()
	}
}
