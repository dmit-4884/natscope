// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"runtime"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func nextChange(t *testing.T, changes <-chan entities.KVChange) entities.KVChange {
	t.Helper()
	select {
	case ch, ok := <-changes:
		require.True(t, ok, "the watch ended early")
		return ch
	case <-time.After(5 * time.Second):
		t.Fatal("no change arrived")
		return entities.KVChange{}
	}
}

func TestWatchKV_ReportsChangesFromNowOnMatchingTheFilter(t *testing.T) {
	t.Parallel()
	c, kv := kvWithKeys(t, "cfg", "orders.0")

	changes, err := c.WatchKV(t.Context(), "cfg", "orders.*")
	require.NoError(t, err)

	_, err = kv.PutString(t.Context(), "users.1", "skipped")
	require.NoError(t, err)
	rev, err := kv.PutString(t.Context(), "orders.1", "new")
	require.NoError(t, err)
	require.NoError(t, kv.Delete(t.Context(), "orders.1"))
	require.NoError(t, kv.Purge(t.Context(), "orders.0"))

	put := nextChange(t, changes)
	assert.Equal(t, "orders.1", put.Key, "keys present before the watch and keys outside the filter are not reported")
	assert.Equal(t, "put", put.Operation)
	assert.Equal(t, rev, put.Revision)
	assert.Equal(t, []byte("new"), put.Value)
	assert.Equal(t, 3, put.Size)
	assert.WithinDuration(t, time.Now(), put.Created, time.Minute)

	del := nextChange(t, changes)
	assert.Equal(t, "orders.1", del.Key)
	assert.Equal(t, "delete", del.Operation)

	purge := nextChange(t, changes)
	assert.Equal(t, "orders.0", purge.Key)
	assert.Equal(t, "purge", purge.Operation)
}

func TestWatchKV_EndsWithTheContext(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg")
	ctx, cancel := context.WithCancel(t.Context())

	changes, err := c.WatchKV(ctx, "cfg", "")
	require.NoError(t, err)
	cancel()

	select {
	case _, ok := <-changes:
		assert.False(t, ok)
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not end")
	}
}

type fakeKVEntry struct {
	key string
	rev uint64
}

func (e fakeKVEntry) Bucket() string                  { return "cfg" }
func (e fakeKVEntry) Key() string                     { return e.key }
func (e fakeKVEntry) Value() []byte                   { return []byte("v") }
func (e fakeKVEntry) Revision() uint64                { return e.rev }
func (e fakeKVEntry) Created() time.Time              { return time.Time{} }
func (e fakeKVEntry) Delta() uint64                   { return 0 }
func (e fakeKVEntry) Operation() jetstream.KeyValueOp { return jetstream.KeyValuePut }

type fakeKeyWatcher struct {
	updates chan jetstream.KeyValueEntry
	stopped atomic.Bool
}

func (w *fakeKeyWatcher) Updates() <-chan jetstream.KeyValueEntry { return w.updates }

func (w *fakeKeyWatcher) Stop() error {
	if w.stopped.CompareAndSwap(false, true) {
		close(w.updates)
	}
	return nil
}

func TestRelayKVChanges_DropsRevisionsFromBeforeTheWatch(t *testing.T) {
	t.Parallel()
	w := &fakeKeyWatcher{updates: make(chan jetstream.KeyValueEntry, 4)}
	w.updates <- fakeKVEntry{key: "old", rev: 7}
	w.updates <- fakeKVEntry{key: "replayed", rev: 10}
	w.updates <- fakeKVEntry{key: "new", rev: 11}
	out := make(chan entities.KVChange, 4)
	ctx, cancel := context.WithCancel(t.Context())

	go relayKVChanges(ctx, "cfg", 10, w, out)

	got := nextChange(t, out)
	assert.Equal(t, "new", got.Key, "a consumer reset replays the stream; revisions up to the start are not changes")
	cancel()
	for range out {
	}
	assert.True(t, w.stopped.Load(), "the watcher is stopped once the relay ends")
}

func TestWatchKV_LeavesNoGoroutineBehindWhenCancelledDuringABurst(t *testing.T) {
	c, kv := kvWithKeys(t, "cfg")
	before := runtime.NumGoroutine()
	ctx, cancel := context.WithCancel(t.Context())

	_, err := c.WatchKV(ctx, "cfg", "")
	require.NoError(t, err)
	for i := range 600 {
		_, err := kv.PutString(t.Context(), "k"+strconv.Itoa(i), "v")
		require.NoError(t, err)
	}
	cancel()

	require.Eventually(t, func() bool { return runtime.NumGoroutine() <= before+2 }, 10*time.Second, 50*time.Millisecond,
		"goroutines: before %d, now %d", before, runtime.NumGoroutine())
}

func TestWatchKV_RefusesABadFilterAndAMissingBucket(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg")

	_, err := c.WatchKV(t.Context(), "cfg", "a.>.b")
	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)

	_, err = c.WatchKV(t.Context(), "missing", "")
	require.ErrorIs(t, err, errs.ErrBucketNotFound)
}
