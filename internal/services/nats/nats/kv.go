// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
)

// ListKVBuckets returns all KeyValue buckets in the connection.
func (s *Service) ListKVBuckets(ctx context.Context, connectionID string) ([]entities.KVBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ListKVBuckets(ctx)
}

// CreateKVBucket creates a new KeyValue bucket.
func (s *Service) CreateKVBucket(
	ctx context.Context,
	connectionID string,
	config entities.KVBucketConfig,
) (*entities.KVBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.CreateKVBucket(ctx, config)
}

// UpdateKVBucket applies new settings to a bucket and keeps every other stream setting.
func (s *Service) UpdateKVBucket(
	ctx context.Context,
	connectionID string,
	bucket string,
	settings entities.KVBucketSettings,
) (*entities.KVBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.UpdateKVBucket(ctx, bucket, settings)
}

// PurgeKVBucket removes every key and revision of a bucket, leaving no delete markers.
func (s *Service) PurgeKVBucket(ctx context.Context, connectionID string, bucket string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.PurgeKVBucket(ctx, bucket)
}

// DeleteKVBucket deletes a KeyValue bucket and all its data.
func (s *Service) DeleteKVBucket(ctx context.Context, connectionID string, bucket string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteKVBucket(ctx, bucket)
}

// GetKVBucket returns information about a specific KeyValue bucket.
func (s *Service) GetKVBucket(ctx context.Context, connectionID string, bucket string) (*entities.KVBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetKVBucket(ctx, bucket)
}

// ListKVKeys returns up to query.Limit keys of a bucket matching query.Filter.
func (s *Service) ListKVKeys(
	ctx context.Context,
	connectionID string,
	bucket string,
	query entities.KVKeysQuery,
) (entities.KVKeyList, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return entities.KVKeyList{}, err
	}
	return c.ListKVKeys(ctx, bucket, query)
}

// WatchKV reports every change from now on to keys matching filter; the channel closes when the watch ends.
func (s *Service) WatchKV(ctx context.Context, connectionID string, bucket string, filter string) (<-chan entities.KVChange, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.WatchKV(ctx, bucket, filter)
}

// GetKVKey returns the value and metadata for a key in a KeyValue bucket.
func (s *Service) GetKVKey(
	ctx context.Context,
	connectionID string,
	bucket string,
	key string,
) (*entities.KVEntry, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetKVKey(ctx, bucket, key)
}

// GetKVKeyHistory returns the stored revisions of a key, oldest first.
func (s *Service) GetKVKeyHistory(
	ctx context.Context,
	connectionID string,
	bucket string,
	key string,
) ([]entities.KVEntry, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetKVKeyHistory(ctx, bucket, key)
}

// PutKVKey stores a value for a key in a KeyValue bucket.
func (s *Service) PutKVKey(
	ctx context.Context,
	connectionID string,
	bucket string,
	key string,
	value []byte,
	expectedRevision uint64,
) (uint64, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return 0, err
	}
	return c.PutKVKey(ctx, bucket, key, value, expectedRevision)
}

// CreateKVKey creates a key that must not exist yet; a non-zero ttl expires it after that long.
func (s *Service) CreateKVKey(
	ctx context.Context,
	connectionID string,
	bucket string,
	key string,
	value []byte,
	ttl time.Duration,
) (uint64, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return 0, err
	}
	return c.CreateKVKey(ctx, bucket, key, value, ttl)
}

// DeleteKVKey deletes a key from a KeyValue bucket.
func (s *Service) DeleteKVKey(ctx context.Context, connectionID string, bucket string, key string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteKVKey(ctx, bucket, key)
}

// PurgeKVKey purges all revisions of a key from a KeyValue bucket.
func (s *Service) PurgeKVKey(ctx context.Context, connectionID string, bucket string, key string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.PurgeKVKey(ctx, bucket, key)
}
