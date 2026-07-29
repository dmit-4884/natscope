// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSourceLocks(t *testing.T) {
	t.Run("same source serialized", func(t *testing.T) {
		var l sourceLocks
		var active, maxActive int32
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				unlock := l.lock("src-1")
				defer unlock()
				cur := atomic.AddInt32(&active, 1)
				for {
					old := atomic.LoadInt32(&maxActive)
					if cur <= old || atomic.CompareAndSwapInt32(&maxActive, old, cur) {
						break
					}
				}
				time.Sleep(2 * time.Millisecond)
				atomic.AddInt32(&active, -1)
			}()
		}
		wg.Wait()
		assert.EqualValues(t, 1, maxActive, "concurrent compiles of one source must serialize")
	})

	t.Run("different sources run concurrently", func(t *testing.T) {
		var l sourceLocks
		releaseA := make(chan struct{})
		gotB := make(chan struct{})
		go func() {
			unlock := l.lock("a")
			defer unlock()
			<-releaseA
		}()
		time.Sleep(5 * time.Millisecond)
		go func() {
			unlock := l.lock("b")
			defer unlock()
			close(gotB)
		}()
		select {
		case <-gotB:
		case <-time.After(time.Second):
			t.Fatal("source b must not wait on source a's lock")
		}
		close(releaseA)
	})

	t.Run("idle entries are dropped", func(t *testing.T) {
		var l sourceLocks
		for range 100 {
			l.lock("src-1")()
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		assert.Empty(t, l.m, "a source that is no longer being compiled must not retain an entry")
	})

	t.Run("entry survives while held", func(t *testing.T) {
		var l sourceLocks
		unlock := l.lock("src-1")

		l.mu.Lock()
		held := len(l.m)
		l.mu.Unlock()
		assert.Equal(t, 1, held)

		unlock()

		l.mu.Lock()
		defer l.mu.Unlock()
		assert.Empty(t, l.m)
	})

	t.Run("waiters keep the entry alive", func(t *testing.T) {
		var l sourceLocks
		unlock := l.lock("src-1")

		waiting := make(chan func(), 1)
		go func() { waiting <- l.lock("src-1") }()

		// Give the waiter time to register its refcount before releasing.
		time.Sleep(20 * time.Millisecond)
		unlock()

		secondUnlock := <-waiting
		l.mu.Lock()
		stillThere := len(l.m)
		l.mu.Unlock()
		assert.Equal(t, 1, stillThere, "the entry must outlive the first holder while a waiter holds it")

		secondUnlock()
		l.mu.Lock()
		defer l.mu.Unlock()
		assert.Empty(t, l.m)
	})
}
