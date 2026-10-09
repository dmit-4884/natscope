// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// kvKeyPattern mirrors nats.go's key charset (jetstream/kv.go validKeyRe).
var kvKeyPattern = regexp.MustCompile(`^[-/_=.a-zA-Z0-9]+$`)

// kvStreamPrefix names the stream behind a bucket: KV_<bucket>.
const kvStreamPrefix = "KV_"

// validateKVKey rejects empty path segments and wildcards with nats.go's "invalid key" error.
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

// wrapBucketErr wraps bucket-level KV/Object errors, preferring the bucket sentinel over the stream one
// and mapping jetstream.ErrBadBucket to errs.ErrNotAKVOrObjectBucket.
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
		converter.WithIgnoreFields("TTL", "LimitMarkerTTL", "Mirror", "Sources", "Republish"),
	)
	kvConfig.TTL = config.TTL
	kvConfig.LimitMarkerTTL = config.LimitMarkerTTL
	if config.LimitMarkerTTL > 0 {
		if err := c.requireFeatures(ctx, featMessageTTL); err != nil {
			return nil, err
		}
	}
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

// DeleteKVBucket deletes a KeyValue bucket and all its data after confirming the stream is a KV bucket.
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

// PurgeKVBucket removes every key and every revision of a bucket in one stream purge after confirming the stream is a
// KV bucket. It leaves no delete markers, so watchers of the bucket are not told.
func (c *Client) PurgeKVBucket(ctx context.Context, bucket string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	if _, err := c.jetStream.KeyValue(ctx, bucket); err != nil {
		return wrapBucketErr(err)
	}
	stream, err := c.jetStream.Stream(ctx, kvStreamPrefix+bucket)
	if err != nil {
		return wrapBucketErr(err)
	}
	if err := stream.Purge(ctx); err != nil {
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
		return nil, wrapBucketErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	result := toKVBucketInfo(status)
	return &result, nil
}

// defaultKVKeysLimit caps a key listing that names no limit.
const defaultKVKeysLimit = 1000

// ListKVKeys returns up to query.Limit keys matching query.Filter. The server applies the filter, and reading stops
// once one key past the limit has arrived. The listing runs an ephemeral consumer, so a missing CONSUMER.CREATE perm
// surfaces only as a timeout. An empty bucket returns an empty list.
func (c *Client) ListKVKeys(ctx context.Context, bucket string, query entities.KVKeysQuery) (entities.KVKeyList, error) {
	filter := strings.TrimSpace(query.Filter)
	if filter != "" {
		if err := natsutil.ValidateSubjectPattern(filter); err != nil {
			return entities.KVKeyList{}, err
		}
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultKVKeysLimit
	}

	kvCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	kv, err := c.jetStream.KeyValue(kvCtx, bucket)
	if err != nil {
		return entities.KVKeyList{}, wrapErr(err)
	}

	var filters []string
	if filter != "" {
		filters = []string{filter}
	}
	lister, err := kv.ListKeysFiltered(kvCtx, filters...)
	if err != nil {
		return entities.KVKeyList{}, wrapErr(err)
	}
	defer func() { _ = lister.Stop() }()

	list := entities.KVKeyList{Keys: []string{}}
	seen := make(map[string]struct{})
	for key := range lister.Keys() {
		if _, dup := seen[key]; dup {
			continue
		}
		if len(list.Keys) == limit {
			list.Truncated = true
			return list, nil
		}
		seen[key] = struct{}{}
		list.Keys = append(list.Keys, key)
	}
	if err := kvCtx.Err(); err != nil {
		if asyncErr := c.takeAsyncError(browseAsyncErrorWait); asyncErr != nil {
			return entities.KVKeyList{}, asyncErr
		}
		return entities.KVKeyList{}, wrapErr(err)
	}
	return list, nil
}

// WatchKV reports every change from now on to the keys of a bucket matching filter, a NATS pattern over key names
// (empty watches every key). The channel closes when ctx ends or the connection drops the watch.
func (c *Client) WatchKV(ctx context.Context, bucket, filter string) (<-chan entities.KVChange, error) {
	filter = strings.TrimSpace(filter)
	if filter != "" {
		if err := natsutil.ValidateSubjectPattern(filter); err != nil {
			return nil, err
		}
	}

	lookupCtx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()
	kv, err := c.jetStream.KeyValue(lookupCtx, bucket)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	var keys []string
	if filter != "" {
		keys = []string{filter}
	}
	watcher, err := kv.WatchFiltered(ctx, keys, jetstream.UpdatesOnly())
	if err != nil {
		return nil, wrapErr(err)
	}

	out := make(chan entities.KVChange)
	go func() {
		defer panics.Handle(ctx)
		defer close(out)
		defer func() { _ = watcher.Stop() }()
		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-watcher.Updates():
				if !ok {
					return
				}
				if entry == nil {
					continue
				}
				e := toKVEntry(bucket, entry)
				change := entities.KVChange{
					Key:       e.Key,
					Operation: e.Operation,
					Revision:  e.Revision,
					Created:   e.Created,
					Value:     entry.Value(),
					Size:      len(entry.Value()),
				}
				select {
				case out <- change:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out, nil
}

// GetKVKey returns the value and metadata for a key in a KeyValue bucket.
func (c *Client) GetKVKey(ctx context.Context, bucket string, key string) (*entities.KVEntry, error) {
	if err := validateNATSSubjectLength("key", key); err != nil {
		return nil, wrapErr(err)
	}
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
	result.TTL = c.kvEntryTTL(kvCtx, bucket, key, entry.Revision())
	return &result, nil
}

// kvEntryTTL reads the Nats-TTL header of a key revision. It is zero when the bucket allows no TTL per key, the
// revision carries none, or the revision cannot be read: the TTL only adds to an entry that was already read.
func (c *Client) kvEntryTTL(ctx context.Context, bucket, key string, revision uint64) time.Duration {
	stream, err := c.jetStream.Stream(ctx, kvStreamPrefix+bucket)
	if err != nil || !stream.CachedInfo().Config.AllowMsgTTL {
		return 0
	}
	msg, err := stream.GetMsg(ctx, revision)
	if err != nil || !strings.HasSuffix(msg.Subject, "."+key) {
		return 0
	}
	return parseMsgTTL(msg.Header.Get(jetstream.MsgTTLHeader))
}

// parseMsgTTL reads a Nats-TTL value the way the server does: a Go duration or whole seconds; "never" and
// anything unreadable mean no expiry.
func parseMsgTTL(value string) time.Duration {
	if d, err := time.ParseDuration(value); err == nil {
		return d
	}
	if secs, err := strconv.Atoi(value); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// minKeyTTL is the shortest TTL the server accepts on a message.
const minKeyTTL = time.Second

// CreateKVKey creates a key that must not exist yet. A non-zero ttl expires the key after that long; the bucket must
// allow a TTL per key.
func (c *Client) CreateKVKey(ctx context.Context, bucket string, key string, value []byte, ttl time.Duration) (uint64, error) {
	if err := validateNATSSubjectLength("key", key); err != nil {
		return 0, wrapErr(err)
	}
	if err := validateKVKey(key); err != nil {
		return 0, err
	}
	if ttl != 0 && ttl < minKeyTTL {
		return 0, &errs.NATSValidationError{Description: "a key TTL must be at least 1s"}
	}

	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	var opts []jetstream.KVCreateOpt
	if ttl > 0 {
		if err := c.requireFeatures(ctx, featMessageTTL); err != nil {
			return 0, err
		}
		opts = append(opts, jetstream.KeyTTL(ttl))
	}

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return 0, wrapErr(err)
	}

	revision, err := kv.Create(ctx, key, value, opts...)
	switch {
	case errors.Is(err, jetstream.ErrKeyExists):
		return 0, &errs.NATSValidationError{Description: fmt.Sprintf("key %q already exists; a TTL can only be set on a new key", key), Cause: err}
	case isAPIErrorCode(err, jsErrCodeMessageTTLDisabled):
		return 0, &errs.NATSValidationError{
			Description: fmt.Sprintf("bucket %q does not allow a TTL per key; turn on the key TTL marker in its settings", bucket),
			Cause:       err,
		}
	case err != nil:
		return 0, wrapErr(err)
	}
	return revision, nil
}

// GetKVKeyHistory returns the stored revisions of a key, oldest first,
// bounded by the bucket's history depth. History() creates an ephemeral
// consumer, so a missing CONSUMER.CREATE perm surfaces only as a timeout.
func (c *Client) GetKVKeyHistory(ctx context.Context, bucket string, key string) ([]entities.KVEntry, error) {
	if err := validateNATSSubjectLength("key", key); err != nil {
		return nil, wrapErr(err)
	}
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
	if err := validateNATSSubjectLength("key", key); err != nil {
		return 0, wrapErr(err)
	}
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
	if err := validateNATSSubjectLength("key", key); err != nil {
		return wrapErr(err)
	}
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
	if err := validateNATSSubjectLength("key", key); err != nil {
		return wrapErr(err)
	}
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
		Bucket:         status.Bucket(),
		Description:    cfg.Description,
		Values:         status.Values(),
		Bytes:          status.Bytes(),
		History:        uint8(history),
		TTL:            status.TTL(),
		Storage:        storage,
		Replicas:       replicas,
		IsCompressed:   status.IsCompressed(),
		Metadata:       status.Metadata(),
		MaxValueSize:   cfg.MaxValueSize,
		MaxBytes:       cfg.MaxBytes,
		LimitMarkerTTL: cfg.LimitMarkerTTL,
	}
}

// kvDuplicateWindow mirrors nats.go: two minutes, or the bucket TTL when that is shorter.
func kvDuplicateWindow(ttl time.Duration) time.Duration {
	const window = 2 * time.Minute
	if ttl > 0 && ttl < window {
		return ttl
	}
	return window
}

// UpdateKVBucket applies new settings to a bucket's stream and keeps every other stream setting as it is.
func (c *Client) UpdateKVBucket(ctx context.Context, bucket string, settings entities.KVBucketSettings) (*entities.KVBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, kvOperationTimeout)
	defer cancel()

	_ = normalizer.Normalize(&settings) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	kv, err := c.jetStream.KeyValue(ctx, bucket)
	if err != nil {
		return nil, wrapBucketErr(err)
	}
	stream, err := c.jetStream.Stream(ctx, kvStreamPrefix+bucket)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	cfg := stream.CachedInfo().Config
	if cfg.SubjectDeleteMarkerTTL > 0 && settings.LimitMarkerTTL <= 0 {
		return nil, &errs.NATSValidationError{Description: "per-key TTL cannot be turned off once a bucket allows it"}
	}
	cfg.Description = settings.Description
	cfg.MaxMsgsPerSubject = int64(max(settings.History, 1))
	cfg.MaxAge = settings.TTL
	cfg.Duplicates = kvDuplicateWindow(settings.TTL)
	cfg.MaxMsgSize = settings.MaxValueSize
	if cfg.MaxMsgSize <= 0 {
		cfg.MaxMsgSize = -1
	}
	cfg.MaxBytes = settings.MaxBytes
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = -1
	}
	cfg.Replicas = max(settings.Replicas, 1)
	cfg.Compression = jetstream.NoCompression
	if settings.Compression {
		cfg.Compression = jetstream.S2Compression
	}
	cfg.SubjectDeleteMarkerTTL = settings.LimitMarkerTTL
	if settings.LimitMarkerTTL > 0 {
		cfg.AllowMsgTTL = true
	}
	cfg.Metadata = settings.Metadata

	if err := c.requireFeatures(ctx, streamConfigFeatures(cfg)...); err != nil {
		return nil, err
	}
	if _, err := c.jetStream.UpdateStream(ctx, cfg); err != nil {
		return nil, wrapErr(err)
	}

	status, err := kv.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}
	result := toKVBucketInfo(status)
	return &result, nil
}
