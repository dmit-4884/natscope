// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// ListKVBuckets returns all KeyValue buckets in the connection.
func (c *Client) ListKVBuckets(ctx context.Context) ([]entities.KVBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	var buckets []entities.KVBucketInfo //nolint:prealloc
	lister := c.jetStream.KeyValueStores(ctx)
	for status := range lister.Status() {
		buckets = append(buckets, toKVBucketInfo(status))
	}

	if err := lister.Error(); err != nil {
		return nil, wrapErr(err)
	}

	return buckets, nil
}

// CreateKVBucket creates a new KeyValue bucket.
func (c *Client) CreateKVBucket(ctx context.Context, config entities.KVBucketConfig) (*entities.KVBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	kvConfig := converter.Convert(config, &jetstream.KeyValueConfig{},
		converter.WithIgnoreFields("TTL", "Mirror", "Sources", "Republish"),
	)
	kvConfig.TTL = config.TTL
	if config.Mirror != nil {
		kvConfig.Mirror = converter.Convert(config.Mirror, &jetstream.StreamSource{})
	}
	for _, src := range config.Sources {
		kvConfig.Sources = append(kvConfig.Sources, converter.Convert(src, &jetstream.StreamSource{}))
	}
	if config.Republish != nil {
		kvConfig.RePublish = converter.Convert(config.Republish, &jetstream.RePublish{},
			converter.WithFieldMappings(map[string]string{"Src": "Source", "Dest": "Destination"}),
		)
	}
	kv, err := c.jetStream.CreateKeyValue(ctx, *kvConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toKVBucketInfo(status)
	return &result, nil
}

// DeleteKVBucket deletes a KeyValue bucket and all its data.
func (c *Client) DeleteKVBucket(ctx context.Context, bucket string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	if err := c.jetStream.DeleteKeyValue(ctx, bucket); err != nil {
		return wrapErr(err)
	}

	return nil
}

// GetKVBucket returns information about a specific KeyValue bucket.
func (c *Client) GetKVBucket(ctx context.Context, bucket string) (*entities.KVBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toKVBucketInfo(status)
	return &result, nil
}

// ListKVKeys returns all keys in a bucket. Keys() creates an ephemeral
// consumer, so a missing CONSUMER.CREATE perm surfaces only as a timeout.
func (c *Client) ListKVKeys(ctx context.Context, bucket string) ([]string, error) {
	kvCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(kvCtx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	keys, err := kv.Keys(kvCtx)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
			if asyncErr := c.takeAsyncError(browseAsyncErrorWait); asyncErr != nil {
				return nil, asyncErr
			}
		}
		return nil, wrapErr(err)
	}

	return keys, nil
}

// GetKVKey returns the value and metadata for a key in a KeyValue bucket.
func (c *Client) GetKVKey(ctx context.Context, bucket string, key string) (*entities.KVEntry, error) {
	kvCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(kvCtx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	entry, err := kv.Get(kvCtx, key)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toKVEntry(bucket, entry)
	return &result, nil
}

// GetKVKeyHistory returns the stored revisions of a key, oldest first,
// bounded by the bucket's history depth. History() creates an ephemeral
// consumer, so a missing CONSUMER.CREATE perm surfaces only as a timeout.
func (c *Client) GetKVKeyHistory(ctx context.Context, bucket string, key string) ([]entities.KVEntry, error) {
	kvCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(kvCtx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	entries, err := kv.History(kvCtx, key)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, nats.ErrTimeout) {
			if asyncErr := c.takeAsyncError(browseAsyncErrorWait); asyncErr != nil {
				return nil, asyncErr
			}
		}
		return nil, wrapErr(err)
	}

	out := make([]entities.KVEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, toKVEntry(bucket, e))
	}
	return out, nil
}

// toKVEntry maps a jetstream entry to the transport entity (value
// base64-encoded, operation lowercased).
func toKVEntry(bucket string, entry jetstream.KeyValueEntry) entities.KVEntry {
	var op string
	switch entry.Operation() {
	case jetstream.KeyValuePut:
		op = "put"
	case jetstream.KeyValueDelete:
		op = "delete"
	case jetstream.KeyValuePurge:
		op = "purge"
	}

	return entities.KVEntry{
		Bucket:    bucket,
		Key:       entry.Key(),
		Value:     base64.StdEncoding.EncodeToString(entry.Value()),
		Revision:  entry.Revision(),
		Created:   entry.Created(),
		Operation: op,
	}
}

// PutKVKey stores a value for a key in a KeyValue bucket. When expectedRevision
// is non-zero the write is a compare-and-swap: it fails unless the key's current
// revision matches, so concurrent writers can't silently clobber each other.
func (c *Client) PutKVKey(ctx context.Context, bucket string, key string, value []byte, expectedRevision uint64) (uint64, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return 0, wrapErr(err)
	}

	var revision uint64
	if expectedRevision > 0 {
		revision, err = kv.Update(ctx, key, value, expectedRevision)
	} else {
		revision, err = kv.Put(ctx, key, value)
	}
	if err != nil {
		return 0, wrapErr(err)
	}

	return revision, nil
}

// DeleteKVKey deletes a key from a KeyValue bucket.
func (c *Client) DeleteKVKey(ctx context.Context, bucket string, key string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return wrapErr(err)
	}

	if err := kv.Delete(ctx, key); err != nil {
		return wrapErr(err)
	}

	return nil
}

// PurgeKVKey purges all revisions of a key from a KeyValue bucket.
func (c *Client) PurgeKVKey(ctx context.Context, bucket string, key string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return wrapErr(err)
	}

	if err := kv.Purge(ctx, key); err != nil {
		return wrapErr(err)
	}

	return nil
}

// maxKVHistory is the depth the server accepts; History is uint8 on the wire.
const maxKVHistory = 64

func toKVBucketInfo(status jetstream.KeyValueStatus) entities.KVBucketInfo {
	cfg := status.Config()

	storage := entities.StorageFile
	if cfg.Storage == jetstream.MemoryStorage {
		storage = entities.StorageMemory
	}

	replicas := cfg.Replicas
	if replicas < 1 {
		replicas = 1
	}

	history := status.History()
	if history < 0 {
		history = 0
	}
	if history > maxKVHistory {
		history = maxKVHistory
	}

	return entities.KVBucketInfo{
		Bucket:       status.Bucket(),
		Values:       status.Values(),
		Bytes:        status.Bytes(),
		History:      uint8(history),
		TTL:          status.TTL(),
		Storage:      storage,
		Replicas:     replicas,
		IsCompressed: status.IsCompressed(),
		Metadata:     status.Metadata(),
	}
}
