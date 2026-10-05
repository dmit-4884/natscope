// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsclient

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// readOnlyClient serves reads from the wrapped client and refuses every write
// with [errs.ErrConnectionReadOnly]. Only the read-only interfaces are
// embedded, so a method added to a write or mixed interface does not compile
// until it is classified here.
type readOnlyClient struct {
	ConnectionInfo
	StreamReader
	Subscriber
	ServiceDiscoverer
	StatsReader

	inner Client
}

var _ Client = (*readOnlyClient)(nil)

func newReadOnlyClient(c Client) *readOnlyClient {
	return &readOnlyClient{
		ConnectionInfo:    c,
		StreamReader:      c,
		Subscriber:        c,
		ServiceDiscoverer: c,
		StatsReader:       c,
		inner:             c,
	}
}

func (r *readOnlyClient) CreateStream(context.Context, entities.StreamCreateRequest) (*entities.StreamInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) UpdateStream(context.Context, string, entities.StreamUpdateRequest) (*entities.StreamInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteStream(context.Context, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) PurgeStream(context.Context, string, entities.StreamPurgeRequest) (uint64, error) {
	return 0, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) SealStream(context.Context, string) (*entities.StreamInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteMessage(context.Context, string, uint64, bool) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) CreateConsumer(context.Context, string, entities.ConsumerCreateRequest) (*entities.ConsumerInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) UpdateConsumer(context.Context, string, string, entities.ConsumerUpdateRequest) (*entities.ConsumerInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteConsumer(context.Context, string, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) PauseConsumer(context.Context, string, string, string) (*entities.ConsumerPauseResponse, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) ResumeConsumer(context.Context, string, string) (*entities.ConsumerPauseResponse, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) ResetConsumer(context.Context, string, string, *uint64) (*entities.ConsumerResetResponse, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) UnpinConsumer(context.Context, string, string, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) Publish(context.Context, string, []byte, map[string]string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) PublishToStream(context.Context, string, []byte, map[string]string) (*entities.PubAck, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) Request(context.Context, string, []byte, map[string]string) (*entities.Reply, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) ListKVBuckets(ctx context.Context) ([]entities.KVBucketInfo, error) {
	return r.inner.ListKVBuckets(ctx)
}

func (r *readOnlyClient) CreateKVBucket(context.Context, entities.KVBucketConfig) (*entities.KVBucketInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteKVBucket(context.Context, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) GetKVBucket(ctx context.Context, bucket string) (*entities.KVBucketInfo, error) {
	return r.inner.GetKVBucket(ctx, bucket)
}

func (r *readOnlyClient) ListKVKeys(ctx context.Context, bucket string) ([]string, error) {
	return r.inner.ListKVKeys(ctx, bucket)
}

func (r *readOnlyClient) GetKVKey(ctx context.Context, bucket, key string) (*entities.KVEntry, error) {
	return r.inner.GetKVKey(ctx, bucket, key)
}

func (r *readOnlyClient) GetKVKeyHistory(ctx context.Context, bucket, key string) ([]entities.KVEntry, error) {
	return r.inner.GetKVKeyHistory(ctx, bucket, key)
}

func (r *readOnlyClient) PutKVKey(context.Context, string, string, []byte, uint64) (uint64, error) {
	return 0, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteKVKey(context.Context, string, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) PurgeKVKey(context.Context, string, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) ListObjectBuckets(ctx context.Context) ([]entities.ObjectBucketInfo, error) {
	return r.inner.ListObjectBuckets(ctx)
}

func (r *readOnlyClient) CreateObjectBucket(context.Context, entities.ObjectBucketConfig) (*entities.ObjectBucketInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteObjectBucket(context.Context, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) GetObjectBucket(ctx context.Context, bucket string) (*entities.ObjectBucketInfo, error) {
	return r.inner.GetObjectBucket(ctx, bucket)
}

func (r *readOnlyClient) ListObjects(ctx context.Context, bucket string) ([]*entities.ObjectInfo, error) {
	return r.inner.ListObjects(ctx, bucket)
}

func (r *readOnlyClient) GetObject(ctx context.Context, bucket, name string) ([]byte, *entities.ObjectInfo, error) {
	return r.inner.GetObject(ctx, bucket, name)
}

func (r *readOnlyClient) PutObject(context.Context, string, entities.ObjectMeta, []byte) (*entities.ObjectInfo, error) {
	return nil, errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) DeleteObject(context.Context, string, string) error {
	return errs.ErrConnectionReadOnly
}

func (r *readOnlyClient) SealObjectBucket(context.Context, string) error {
	return errs.ErrConnectionReadOnly
}
