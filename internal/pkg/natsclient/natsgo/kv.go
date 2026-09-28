// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// kvKeyPattern mirrors nats.go's own key charset (jetstream/kv.go validKeyRe)
// so validateKVKey rejects nothing a successful nats.go call would accept.
var kvKeyPattern = regexp.MustCompile(`^[-/_=.a-zA-Z0-9]+$`)

// validateKVKey rejects key shapes nats.go's own validation lets through
// inconsistently: empty path segments such as "a..b" pass nats.go's keyValid
// (only leading/trailing dots are checked), and wildcards pass History's
// Watch-based validator even though every other key RPC rejects them (see
// QA-062, QA-064). A rejected key never reaches the wire, so the error
// carries the same "nats: invalid key" text nats.go uses for the cases it
// does catch.
func validateKVKey(key string) error {
	if key != "" && key[0] != '.' && key[len(key)-1] != '.' && !strings.Contains(key, "..") &&
		kvKeyPattern.MatchString(key) {
		return nil
	}
	return &errs.NATSValidationError{
		Description: jetstream.ErrInvalidKey.Error(),
		Cause:       jetstream.ErrInvalidKey,
	}
}

// wrapBucketErr wraps errors from KV/Object bucket-level operations
// (Create/Get/Delete/Seal). nats.go's own error chains for a missing bucket
// join both jetstream.ErrBucketNotFound and jetstream.ErrStreamNotFound, and
// natsSentinelMap (natsgo/errors.go) lists the stream sentinel first, so
// plain wrapErr reports the stream-level reason instead of the bucket-level
// one a caller actually asked about (see QA-124). It also translates
// jetstream.ErrBadBucket, which wrapErr never maps, so a stream that merely
// shares a KV_/OBJ_ name refuses instead of silently being deleted or sealed
// (see QA-066).
func wrapBucketErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, jetstream.ErrBadBucket):
		return errors.Join(errs.ErrNotAKVOrObjectBucket, err)
	case errors.Is(err, jetstream.ErrBucketNotFound):
		return errors.Join(errs.ErrBucketNotFound, err)
	case errors.Is(err, jetstream.ErrBucketExists):
		return errors.Join(errs.ErrBucketExists, err)
	}
	return wrapErr(err)
}

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
		kvConfig.RePublish = converter.Convert(config.Republish, &jetstream.RePublish{}, srcDestToJetStream)
	}
	kv, err := c.jetStream.CreateKeyValue(ctx, *kvConfig)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toKVBucketInfo(status)
	return &result, nil
}

// DeleteKVBucket deletes a KeyValue bucket and all its data. nats.go's
// DeleteKeyValue skips the sanity check KeyValue() itself does (history depth
// > 0), so a stream merely named KV_<bucket> would otherwise be deleted
// outright; KeyValue() first confirms it's really a KV bucket (see QA-066).
func (c *Client) DeleteKVBucket(ctx context.Context, bucket string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	if _, err := c.jetStream.KeyValue(ctx, bucket); err != nil {
		return wrapBucketErr(err)
	}

	if err := c.jetStream.DeleteKeyValue(ctx, bucket); err != nil {
		return wrapBucketErr(err)
	}

	return nil
}

// GetKVBucket returns information about a specific KeyValue bucket.
func (c *Client) GetKVBucket(ctx context.Context, bucket string) (*entities.KVBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	result := toKVBucketInfo(status)
	return &result, nil
}

// ListKVKeys returns all keys in a bucket. Keys() creates an ephemeral
// consumer, so a missing CONSUMER.CREATE perm surfaces only as a timeout. An
// empty bucket returns an empty slice rather than jetstream.ErrNoKeysFound
// (see QA-065), matching every other empty-collection response in this API.
func (c *Client) ListKVKeys(ctx context.Context, bucket string) ([]string, error) {
	kvCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(kvCtx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	keys, err := kv.Keys(kvCtx)
	if errors.Is(err, jetstream.ErrNoKeysFound) {
		return []string{}, nil
	}
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
	if err := validateKVKey(key); err != nil {
		return nil, err
	}

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
	if err := validateKVKey(key); err != nil {
		return nil, err
	}

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
	if err := validateKVKey(key); err != nil {
		return 0, err
	}

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
	if err := validateKVKey(key); err != nil {
		return err
	}

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
	if err := validateKVKey(key); err != nil {
		return err
	}

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
		Description:  cfg.Description,
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
