// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package live

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
)

type fakeKVWatch struct {
	natssvc.KVStore
	changes chan entities.KVChange
	err     error
	bucket  string
	filter  string

	mu        sync.Mutex
	bucketErr error
}

func (f *fakeKVWatch) WatchKV(_ context.Context, _, bucket, filter string) (<-chan entities.KVChange, error) {
	f.bucket, f.filter = bucket, filter
	return f.changes, f.err
}

func (f *fakeKVWatch) GetKVBucket(context.Context, string, string) (*entities.KVBucketInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &entities.KVBucketInfo{}, f.bucketErr
}

func (f *fakeKVWatch) setBucketErr(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bucketErr = err
}

type emitted struct {
	mu     sync.Mutex
	events []entities.KVWatchEvent
}

func (e *emitted) emit(ev entities.KVWatchEvent) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.events = append(e.events, ev)
	return nil
}

func (e *emitted) all() []entities.KVChange {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []entities.KVChange
	for _, ev := range e.events {
		out = append(out, ev.Changes...)
	}
	return out
}

func (e *emitted) last() entities.KVWatchEvent {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.events[len(e.events)-1]
}

func kvWatchService(kv *fakeKVWatch) *Service {
	return New(nil, &fakeSubscriber{}, kv, fakeCodec{}, fakeSettings{})
}

func TestWatchKV_BatchesChangesAndCutsLongValues(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{changes: make(chan entities.KVChange)}
	ctx, cancel := context.WithCancel(t.Context())
	out := &emitted{}
	done := make(chan error, 1)
	go func() {
		done <- kvWatchService(kv).WatchKV(ctx, &entities.KVWatchRequest{ConnectionID: "c", Bucket: "cfg", Filter: "a.*"}, out.emit)
	}()

	long := bytes.Repeat([]byte("x"), 2000)
	kv.changes <- entities.KVChange{Key: "a.1", Operation: "put", Value: []byte("v"), Size: 1}
	kv.changes <- entities.KVChange{Key: "a.2", Operation: "put", Value: long, Size: len(long)}

	require.Eventually(t, func() bool { return len(out.all()) == 2 }, 2*time.Second, 20*time.Millisecond)
	got := out.all()
	assert.Equal(t, "a.1", got[0].Key)
	assert.Len(t, got[1].Value, kvWatchValueBytes)
	assert.Equal(t, 2000, got[1].Size, "the size stays the size of the whole value")
	assert.Equal(t, "cfg", kv.bucket)
	assert.Equal(t, "a.*", kv.filter)

	cancel()
	assert.NoError(t, <-done, "a watch the caller ends is not an error")
}

func TestWatchKV_ConfirmsTheWatchWithAnEmptyFirstBatch(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{changes: make(chan entities.KVChange)}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first := make(chan entities.KVWatchEvent, 1)
	go func() {
		_ = kvWatchService(kv).WatchKV(ctx, &entities.KVWatchRequest{Bucket: "cfg"}, func(ev entities.KVWatchEvent) error {
			select {
			case first <- ev:
			default:
			}
			return nil
		})
	}()

	select {
	case ev := <-first:
		assert.Empty(t, ev.Changes)
		assert.False(t, ev.Offline)
	case <-time.After(time.Second):
		t.Fatal("the watch sent nothing before the first change")
	}
}

func TestWatchKV_ReportsAWatchTheConnectionDropped(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{changes: make(chan entities.KVChange, 1)}
	out := &emitted{}
	kv.changes <- entities.KVChange{Key: "a", Operation: "put"}
	close(kv.changes)

	err := kvWatchService(kv).WatchKV(t.Context(), &entities.KVWatchRequest{Bucket: "cfg"}, out.emit)

	require.ErrorIs(t, err, errs.ErrNATSConnectionClosed)
	assert.Len(t, out.all(), 1, "changes seen before the drop still go out")
}

func TestWatchKV_StopsWhenEmitFails(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{changes: make(chan entities.KVChange, 1)}
	kv.changes <- entities.KVChange{Key: "a", Operation: "put"}
	boom := errors.New("client gone")

	err := kvWatchService(kv).WatchKV(t.Context(), &entities.KVWatchRequest{Bucket: "cfg"}, func(entities.KVWatchEvent) error { return boom })

	require.ErrorIs(t, err, boom)
}

func TestWatchKV_PassesOnAWatchThatCannotStart(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{err: errs.ErrBucketNotFound}

	err := kvWatchService(kv).WatchKV(t.Context(), &entities.KVWatchRequest{Bucket: "missing"}, (&emitted{}).emit)

	require.ErrorIs(t, err, errs.ErrBucketNotFound)
}

func TestWatchKV_ReportsTheBucketStreamOfflineAndBack(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		kv := &fakeKVWatch{changes: make(chan entities.KVChange)}
		ctx, cancel := context.WithCancel(t.Context())
		out := &emitted{}
		done := make(chan error, 1)
		go func() {
			done <- kvWatchService(kv).WatchKV(ctx, &entities.KVWatchRequest{Bucket: "cfg"}, out.emit)
		}()

		kv.setBucketErr(&errs.NATSAPIError{Code: 500, ErrorCode: 10118, Description: "stream is offline"})
		time.Sleep(kvHealthInterval + time.Second)
		synctest.Wait()
		assert.True(t, out.last().Offline, "the watch says its bucket's stream is out of reach")

		kv.setBucketErr(nil)
		time.Sleep(kvHealthInterval)
		synctest.Wait()
		assert.False(t, out.last().Offline, "the watch says the stream is back")
		assert.Empty(t, out.last().Changes)

		cancel()
		assert.NoError(t, <-done)
	})
}

func TestWatchKV_StaysOnlineWhenAHealthCheckFailsForAnotherReason(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		kv := &fakeKVWatch{changes: make(chan entities.KVChange)}
		ctx, cancel := context.WithCancel(t.Context())
		out := &emitted{}
		done := make(chan error, 1)
		go func() {
			done <- kvWatchService(kv).WatchKV(ctx, &entities.KVWatchRequest{Bucket: "cfg"}, out.emit)
		}()

		kv.setBucketErr(errs.ErrNATSTimeout)
		time.Sleep(2 * kvHealthInterval)
		synctest.Wait()

		require.Len(t, out.events, 1, "only the confirming frame was sent")
		cancel()
		assert.NoError(t, <-done)
	})
}
