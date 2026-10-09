// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

func kvWithKeys(t *testing.T, bucket string, keys ...string) (*Client, jetstream.KeyValue) {
	t.Helper()
	_, url := jetStreamServer(t)
	nc, err := nats.Connect(url)
	require.NoError(t, err)
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
	require.NoError(t, err)
	kv, err := js.CreateKeyValue(t.Context(), jetstream.KeyValueConfig{Bucket: bucket})
	require.NoError(t, err)
	for _, k := range keys {
		_, err = kv.PutString(t.Context(), k, "v")
		require.NoError(t, err)
	}
	return dialClient(t, url), kv
}

func TestListKVKeys_FiltersOnTheServer(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg", "orders.1", "orders.2", "orders.eu.3", "users.1")

	got, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Filter: "orders.*", Limit: 10})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"orders.1", "orders.2"}, got.Keys)
	assert.False(t, got.Truncated)
}

func TestListKVKeys_EmptyFilterListsEveryKey(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg", "a", "b.c")

	got, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Limit: 10})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a", "b.c"}, got.Keys)
}

func TestListKVKeys_StopsAtTheLimit(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg", "a", "b", "c")

	cut, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, cut.Keys, 2)
	assert.True(t, cut.Truncated, "a third key exists beyond the limit")

	whole, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Limit: 3})
	require.NoError(t, err)
	assert.Len(t, whole.Keys, 3)
	assert.False(t, whole.Truncated, "every key fits")
}

func TestListKVKeys_DefaultLimitAppliesWhenUnset(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg", "a", "b")

	got, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{})

	require.NoError(t, err)
	assert.Len(t, got.Keys, 2)
	assert.False(t, got.Truncated)
}

func TestListKVKeys_SkipsDeletedKeys(t *testing.T) {
	t.Parallel()
	c, kv := kvWithKeys(t, "cfg", "keep", "gone")
	require.NoError(t, kv.Delete(t.Context(), "gone"))

	got, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Limit: 10})

	require.NoError(t, err)
	assert.Equal(t, []string{"keep"}, got.Keys)
}

func TestListKVKeys_EmptyBucket(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg")

	got, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Filter: "x.>", Limit: 10})

	require.NoError(t, err)
	assert.Empty(t, got.Keys)
	assert.NotNil(t, got.Keys)
	assert.False(t, got.Truncated)
}

func TestListKVKeys_RejectsAnInvalidFilter(t *testing.T) {
	t.Parallel()
	c, _ := kvWithKeys(t, "cfg", "a")

	for _, filter := range []string{"a.>.b", ".a", "a b", "a..b"} {
		_, err := c.ListKVKeys(t.Context(), "cfg", entities.KVKeysQuery{Filter: filter, Limit: 10})
		var verr *errs.NATSValidationError
		assert.ErrorAs(t, err, &verr, filter)
	}
}
