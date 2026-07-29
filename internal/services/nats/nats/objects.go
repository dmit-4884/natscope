// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// ListObjectBuckets returns all Object Store buckets in the connection.
func (s *Service) ListObjectBuckets(ctx context.Context, connectionID string) ([]entities.ObjectBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ListObjectBuckets(ctx)
}

// CreateObjectBucket creates a new Object Store bucket.
func (s *Service) CreateObjectBucket(
	ctx context.Context,
	connectionID string,
	config entities.ObjectBucketConfig,
) (*entities.ObjectBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.CreateObjectBucket(ctx, config)
}

// DeleteObjectBucket deletes an Object Store bucket and all its objects.
func (s *Service) DeleteObjectBucket(ctx context.Context, connectionID string, bucket string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteObjectBucket(ctx, bucket)
}

// GetObjectBucket returns information about a specific Object Store bucket.
func (s *Service) GetObjectBucket(
	ctx context.Context,
	connectionID string,
	bucket string,
) (*entities.ObjectBucketInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetObjectBucket(ctx, bucket)
}

// ListObjects returns all objects in an Object Store bucket.
func (s *Service) ListObjects(ctx context.Context, connectionID string, bucket string) ([]*entities.ObjectInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ListObjects(ctx, bucket)
}

// GetObject returns the content and metadata of an object.
func (s *Service) GetObject(
	ctx context.Context,
	connectionID string,
	bucket string,
	name string,
) ([]byte, *entities.ObjectInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, nil, err
	}
	return c.GetObject(ctx, bucket, name)
}

// PutObject stores an object in an Object Store bucket.
func (s *Service) PutObject(
	ctx context.Context,
	connectionID string,
	bucket string,
	meta entities.ObjectMeta,
	data []byte,
) (*entities.ObjectInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.PutObject(ctx, bucket, meta, data)
}

// DeleteObject deletes an object from an Object Store bucket.
func (s *Service) DeleteObject(ctx context.Context, connectionID string, bucket string, name string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteObject(ctx, bucket, name)
}

// SealObjectBucket seals an Object Store bucket, making it read-only.
func (s *Service) SealObjectBucket(ctx context.Context, connectionID string, bucket string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.SealObjectBucket(ctx, bucket)
}
