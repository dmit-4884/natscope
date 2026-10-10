// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// fakeKVStatus is a jetstream.KeyValueStatus whose every field is distinct from
// the defaults toKVBucketInfo used to hardcode.
type fakeKVStatus struct {
	bucket   string
	values   uint64
	history  int64
	ttl      time.Duration
	bytes    uint64
	compress bool
	meta     map[string]string
	cfg      jetstream.KeyValueConfig
}

func (f fakeKVStatus) Bucket() string                   { return f.bucket }
func (f fakeKVStatus) Values() uint64                   { return f.values }
func (f fakeKVStatus) History() int64                   { return f.history }
func (f fakeKVStatus) TTL() time.Duration               { return f.ttl }
func (f fakeKVStatus) BackingStore() string             { return "JetStream" }
func (f fakeKVStatus) Bytes() uint64                    { return f.bytes }
func (f fakeKVStatus) IsCompressed() bool               { return f.compress }
func (f fakeKVStatus) LimitMarkerTTL() time.Duration    { return 0 }
func (f fakeKVStatus) Metadata() map[string]string      { return f.meta }
func (f fakeKVStatus) Config() jetstream.KeyValueConfig { return f.cfg }

var _ jetstream.KeyValueStatus = fakeKVStatus{}

func TestToKVBucketInfo_ReportsRealStorageAndReplicas(t *testing.T) {
	t.Parallel()

	got := toKVBucketInfo(fakeKVStatus{
		bucket:  "cfg",
		values:  7,
		history: 5,
		ttl:     time.Minute,
		bytes:   1024,
		cfg:     jetstream.KeyValueConfig{Storage: jetstream.MemoryStorage, Replicas: 3},
	})

	assert.Equal(t, "cfg", got.Bucket)
	assert.Equal(t, uint64(7), got.Values)
	assert.Equal(t, uint8(5), got.History)
	assert.Equal(t, time.Minute, got.TTL)
	assert.Equal(t, uint64(1024), got.Bytes)
	assert.Equal(t, entities.StorageMemory, got.Storage, "memory buckets must not report file storage")
	assert.Equal(t, 3, got.Replicas, "replica count must come from the bucket config")
}

// TestToKVBucketInfo_ReportsDescription checks that bucket reads return the description.
func TestToKVBucketInfo_ReportsDescription(t *testing.T) {
	t.Parallel()

	got := toKVBucketInfo(fakeKVStatus{
		bucket: "cfg",
		cfg:    jetstream.KeyValueConfig{Description: "qa desc kv1"},
	})

	assert.Equal(t, "qa desc kv1", got.Description)
}

func TestToKVBucketInfo_FileStorage(t *testing.T) {
	t.Parallel()

	got := toKVBucketInfo(fakeKVStatus{
		bucket: "cfg",
		cfg:    jetstream.KeyValueConfig{Storage: jetstream.FileStorage, Replicas: 1},
	})

	assert.Equal(t, entities.StorageFile, got.Storage)
	assert.Equal(t, 1, got.Replicas)
}

func TestToKVBucketInfo_NormalizesDegenerateValues(t *testing.T) {
	t.Parallel()

	t.Run("zero replicas mean one", func(t *testing.T) {
		t.Parallel()
		got := toKVBucketInfo(fakeKVStatus{cfg: jetstream.KeyValueConfig{Replicas: 0}})
		assert.Equal(t, 1, got.Replicas)
	})

	t.Run("history is clamped instead of wrapping", func(t *testing.T) {
		t.Parallel()
		got := toKVBucketInfo(fakeKVStatus{history: 300})
		assert.Equal(t, uint8(maxKVHistory), got.History)
	})

	t.Run("negative history is floored", func(t *testing.T) {
		t.Parallel()
		got := toKVBucketInfo(fakeKVStatus{history: -1})
		assert.Equal(t, uint8(0), got.History)
	})
}

// TestValidateKVKey checks that empty path segments and wildcards are rejected.
func TestValidateKVKey(t *testing.T) {
	t.Parallel()

	valid := []string{"a", "a.b", "a.b.c", "a-b_c=d", "123"}
	for _, key := range valid {
		t.Run("valid/"+key, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, validateKVKey(key))
		})
	}

	invalid := []string{"", ".a", "a.", "a..b", "a...b", "*", ">", "a.*", "a.>", "a b"}
	for _, key := range invalid {
		t.Run("invalid/"+key, func(t *testing.T) {
			t.Parallel()
			err := validateKVKey(key)
			require.Error(t, err)
			assert.ErrorIs(t, err, errs.ErrNATSInvalidArgument)
		})
	}
}

// TestWrapBucketErr checks that the bucket sentinel wins and ErrBadBucket maps to ErrNotAKVOrObjectBucket.
func TestWrapBucketErr(t *testing.T) {
	t.Parallel()

	t.Run("nil in, nil out", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, wrapBucketErr(nil))
	})

	t.Run("bucket-not-found wins over stream-not-found", func(t *testing.T) {
		t.Parallel()
		joined := errors.Join(jetstream.ErrBucketNotFound, jetstream.ErrStreamNotFound)
		err := wrapBucketErr(joined)
		assert.ErrorIs(t, err, errs.ErrBucketNotFound)
	})

	t.Run("bucket-exists wins over stream-name-in-use", func(t *testing.T) {
		t.Parallel()
		joined := errors.Join(jetstream.ErrBucketExists, jetstream.ErrStreamNameAlreadyInUse)
		err := wrapBucketErr(joined)
		assert.ErrorIs(t, err, errs.ErrBucketExists)
	})

	t.Run("translates ErrBadBucket instead of falling through to Internal", func(t *testing.T) {
		t.Parallel()
		err := wrapBucketErr(jetstream.ErrBadBucket)
		assert.ErrorIs(t, err, errs.ErrNotAKVOrObjectBucket)
	})

	t.Run("falls through to wrapErr for anything else", func(t *testing.T) {
		t.Parallel()
		err := wrapBucketErr(jetstream.ErrKeyNotFound)
		assert.ErrorIs(t, err, errs.ErrKeyNotFound)
	})
}
