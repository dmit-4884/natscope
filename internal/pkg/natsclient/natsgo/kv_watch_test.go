// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"testing"
	"time"

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

func TestWatchKV_RefusesABadFilterAndAMissingBucket(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg")

	_, err := c.WatchKV(t.Context(), "cfg", "a.>.b")
	var verr *errs.NATSValidationError
	require.ErrorAs(t, err, &verr)

	_, err = c.WatchKV(t.Context(), "missing", "")
	require.ErrorIs(t, err, errs.ErrBucketNotFound)
}
