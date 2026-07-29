// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bytes"
	"context"
	"errors"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
)

// ListObjectBuckets returns all Object Store buckets in the connection.
func (c *Client) ListObjectBuckets(ctx context.Context) ([]entities.ObjectBucketInfo, error) {
	var buckets []entities.ObjectBucketInfo //nolint:prealloc
	lister := c.jetStream.ObjectStores(ctx)
	for status := range lister.Status() {
		count, err := countObjects(ctx, c.jetStream, status.Bucket())
		if err != nil {
			return nil, wrapErr(err)
		}
		buckets = append(buckets, toObjectBucketInfo(status, count))
	}

	if err := lister.Error(); err != nil {
		return nil, wrapErr(err)
	}

	return buckets, nil
}

// countObjects returns the object count; an empty bucket surfaces as
// jetstream.ErrNoObjectsFound instead of an empty slice, so that's treated as 0.
func countObjects(ctx context.Context, js jetstream.JetStream, bucket string) (uint64, error) {
	obj, err := js.ObjectStore(ctx, bucket)
	if err != nil {
		return 0, err
	}
	list, err := obj.List(ctx)
	if errors.Is(err, jetstream.ErrNoObjectsFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return uint64(len(list)), nil
}

// CreateObjectBucket creates a new Object Store bucket.
func (c *Client) CreateObjectBucket(ctx context.Context, config entities.ObjectBucketConfig) (*entities.ObjectBucketInfo, error) {
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	objConfig := converter.Convert(config, &jetstream.ObjectStoreConfig{},
		converter.WithIgnoreFields("TTL"),
	)
	objConfig.TTL = config.TTL
	obj, err := c.jetStream.CreateObjectStore(ctx, *objConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	status, err := obj.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	// A freshly created bucket has no objects yet.
	result := toObjectBucketInfo(status, 0)
	return &result, nil
}

// DeleteObjectBucket deletes an Object Store bucket and all its objects.
func (c *Client) DeleteObjectBucket(ctx context.Context, bucket string) error {
	if err := c.jetStream.DeleteObjectStore(ctx, bucket); err != nil {
		return wrapErr(err)
	}

	return nil
}

// GetObjectBucket returns information about a specific Object Store bucket.
func (c *Client) GetObjectBucket(ctx context.Context, bucket string) (*entities.ObjectBucketInfo, error) {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	status, err := obj.Status(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	count, err := countObjects(ctx, c.jetStream, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toObjectBucketInfo(status, count)
	return &result, nil
}

// ListObjects returns all objects in an Object Store bucket.
func (c *Client) ListObjects(ctx context.Context, bucket string) ([]*entities.ObjectInfo, error) {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	list, err := obj.List(ctx)
	if err != nil {
		return nil, wrapErr(err)
	}

	return slices.To(list, func(info *jetstream.ObjectInfo) *entities.ObjectInfo {
		return converter.Convert(info, &entities.ObjectInfo{})
	}), nil
}

// GetObject returns the content and metadata of an object.
func (c *Client) GetObject(ctx context.Context, bucket string, name string) ([]byte, *entities.ObjectInfo, error) {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, nil, wrapErr(err)
	}

	data, err := obj.GetBytes(ctx, name)
	if err != nil {
		return nil, nil, wrapErr(err)
	}

	info, err := obj.GetInfo(ctx, name)
	if err != nil {
		return nil, nil, wrapErr(err)
	}

	return data, converter.Convert(info, &entities.ObjectInfo{}), nil
}

// PutObject stores an object in an Object Store bucket.
func (c *Client) PutObject(
	ctx context.Context,
	bucket string,
	meta entities.ObjectMeta,
	data []byte,
) (*entities.ObjectInfo, error) {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	objMeta := converter.Convert(meta, &jetstream.ObjectMeta{})

	info, err := obj.Put(ctx, *objMeta, bytes.NewReader(data))
	if err != nil {
		return nil, wrapErr(err)
	}

	return converter.Convert(info, &entities.ObjectInfo{}), nil
}

// DeleteObject deletes an object from an Object Store bucket.
func (c *Client) DeleteObject(ctx context.Context, bucket string, name string) error {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return wrapErr(err)
	}

	if err := obj.Delete(ctx, name); err != nil {
		return wrapErr(err)
	}

	return nil
}

// SealObjectBucket seals an Object Store bucket, making it read-only.
func (c *Client) SealObjectBucket(ctx context.Context, bucket string) error {
	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return wrapErr(err)
	}

	if err := obj.Seal(ctx); err != nil {
		return wrapErr(err)
	}

	return nil
}

func toObjectBucketInfo(status jetstream.ObjectStoreStatus, objectCount uint64) entities.ObjectBucketInfo {
	return entities.ObjectBucketInfo{
		Bucket:       status.Bucket(),
		Description:  status.Description(),
		Size:         status.Size(),
		Objects:      objectCount,
		Storage:      entities.StorageType(status.Storage()),
		Replicas:     status.Replicas(),
		Sealed:       status.Sealed(),
		IsCompressed: status.IsCompressed(),
		Metadata:     status.Metadata(),
	}
}
