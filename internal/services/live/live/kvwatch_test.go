// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package live

import (
	"bytes"
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
)

type fakeKVWatch struct {
	natssvc.KVStore
	changes chan entities.KVChange
	err     error
	bucket  string
	filter  string
}

func (f *fakeKVWatch) WatchKV(_ context.Context, _, bucket, filter string) (<-chan entities.KVChange, error) {
	f.bucket, f.filter = bucket, filter
	return f.changes, f.err
}

type emitted struct {
	mu      sync.Mutex
	batches [][]entities.KVChange
}

func (e *emitted) emit(batch []entities.KVChange) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.batches = append(e.batches, batch)
	return nil
}

func (e *emitted) all() []entities.KVChange {
	e.mu.Lock()
	defer e.mu.Unlock()
	var out []entities.KVChange
	for _, b := range e.batches {
		out = append(out, b...)
	}
	return out
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
	first := make(chan []entities.KVChange, 1)
	go func() {
		_ = kvWatchService(kv).WatchKV(ctx, &entities.KVWatchRequest{Bucket: "cfg"}, func(batch []entities.KVChange) error {
			select {
			case first <- batch:
			default:
			}
			return nil
		})
	}()

	select {
	case batch := <-first:
		assert.Empty(t, batch)
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

	err := kvWatchService(kv).WatchKV(t.Context(), &entities.KVWatchRequest{Bucket: "cfg"}, func([]entities.KVChange) error { return boom })

	require.ErrorIs(t, err, boom)
}

func TestWatchKV_PassesOnAWatchThatCannotStart(t *testing.T) {
	t.Parallel()
	kv := &fakeKVWatch{err: errs.ErrBucketNotFound}

	err := kvWatchService(kv).WatchKV(t.Context(), &entities.KVWatchRequest{Bucket: "missing"}, (&emitted{}).emit)

	require.ErrorIs(t, err, errs.ErrBucketNotFound)
}
