// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

const (
	// objectOperationTimeout bounds Object Store operations; longer than kvOperationTimeout for larger payloads.
	objectOperationTimeout = 30 * time.Second

	// maxGetObjectBytes caps GetObject's in-memory buffer; matches the transport's maxRequestBytes.
	maxGetObjectBytes = 32 << 20 // 32 MiB

	// objChunksSubjectSuffix and objMetaSubjectSuffix mirror nats.go's Object Store subject layout.
	objChunksSubjectSuffix = ".C.>"
	objMetaSubjectSuffix   = ".M.>"
)

// streamInfoProvider exposes the jetstream.StreamInfo behind *jetstream.ObjectBucketStatus.
type streamInfoProvider interface {
	StreamInfo() *jetstream.StreamInfo
}

// ListObjectBuckets returns all Object Store buckets in the connection.
func (c *Client) ListObjectBuckets(ctx context.Context) ([]entities.ObjectBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

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
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	objConfig := converter.Convert(config, &jetstream.ObjectStoreConfig{},
		converter.WithIgnoreFields("TTL"),
	)
	objConfig.TTL = config.TTL
	obj, err := c.jetStream.CreateObjectStore(ctx, *objConfig)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	status, err := obj.Status(ctx)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	// A freshly created bucket has no objects yet.
	result := toObjectBucketInfo(status, 0)
	return &result, nil
}

// DeleteObjectBucket deletes an Object Store bucket and all its objects once verifyObjectBucket accepts it.
func (c *Client) DeleteObjectBucket(ctx context.Context, bucket string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return wrapBucketErr(err)
	}
	if err := verifyObjectBucket(ctx, obj, bucket); err != nil {
		return err
	}

	if err := c.jetStream.DeleteObjectStore(ctx, bucket); err != nil {
		return wrapBucketErr(err)
	}

	return nil
}

// GetObjectBucket returns information about a specific Object Store bucket.
func (c *Client) GetObjectBucket(ctx context.Context, bucket string) (*entities.ObjectBucketInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	status, err := obj.Status(ctx)
	if err != nil {
		return nil, wrapBucketErr(err)
	}

	count, err := countObjects(ctx, c.jetStream, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	result := toObjectBucketInfo(status, count)
	return &result, nil
}

// ListObjects returns all objects in an Object Store bucket; an empty bucket
// returns an empty slice.
func (c *Client) ListObjects(ctx context.Context, bucket string) ([]*entities.ObjectInfo, error) {
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	list, err := obj.List(ctx)
	if errors.Is(err, jetstream.ErrNoObjectsFound) {
		return []*entities.ObjectInfo{}, nil
	}
	if err != nil {
		return nil, wrapErr(err)
	}

	return slices.To(list, toObjectInfo), nil
}

// GetObject returns an object's content and metadata from a single Get, so the two always match.
// A link is followed to its target; a link to a whole bucket is rejected.
func (c *Client) GetObject(ctx context.Context, bucket string, name string) ([]byte, *entities.ObjectInfo, error) {
	if err := validateNATSSubjectLength("object name", name); err != nil {
		return nil, nil, wrapErr(err)
	}

	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, nil, wrapErr(err)
	}

	result, err := obj.Get(ctx, name)
	if err != nil {
		if errors.Is(err, jetstream.ErrCantGetBucket) {
			return nil, nil, errs.ErrObjectLinkToBucket
		}
		return nil, nil, wrapErr(err)
	}
	defer result.Close() //nolint:errcheck // best-effort close

	info, err := result.Info()
	if err != nil {
		return nil, nil, wrapErr(err)
	}
	if info.Size > maxGetObjectBytes {
		return nil, nil, errs.ErrObjectTooLargeToRetrieve
	}

	data, err := io.ReadAll(result)
	if err != nil {
		return nil, nil, wrapErr(err)
	}

	return data, toObjectInfo(info), nil
}

// checkObjectCapacity refuses a Put that would clearly exceed the bucket's max_bytes before nats.go's
// async Put can replace the current object. It is best-effort: an unreadable status lets the Put proceed.
func checkObjectCapacity(ctx context.Context, obj jetstream.ObjectStore, name string, newSize int64) error {
	status, err := obj.Status(ctx)
	if err != nil {
		return nil //nolint:nilerr // best-effort guard
	}
	provider, ok := status.(streamInfoProvider)
	if !ok {
		return nil
	}
	info := provider.StreamInfo()
	if info == nil || info.Config.MaxBytes <= 0 {
		return nil
	}

	var oldSize int64
	if existing, getErr := obj.GetInfo(ctx, name); getErr == nil && existing != nil {
		oldSize = int64(existing.Size) //nolint:gosec // sizes fit int64
	}

	if int64(info.State.Bytes)-oldSize+newSize <= info.Config.MaxBytes { //nolint:gosec // same
		return nil
	}
	return errs.ErrObjectBucketCapacityExceeded
}

// PutObject stores an object, serializing writes per (bucket, name) and refusing ones that overflow the bucket.
func (c *Client) PutObject(
	ctx context.Context,
	bucket string,
	meta entities.ObjectMeta,
	data []byte,
) (*entities.ObjectInfo, error) {
	if err := validateNATSSubjectLength("object name", meta.Name); err != nil {
		return nil, wrapErr(err)
	}

	unlock := c.lockObjectPut(bucket, meta.Name)
	defer unlock()

	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return nil, wrapErr(err)
	}

	if capErr := checkObjectCapacity(ctx, obj, meta.Name, int64(len(data))); capErr != nil { //nolint:gosec // data is bounded by maxRequestBytes
		return nil, capErr
	}

	objMeta := converter.Convert(meta, &jetstream.ObjectMeta{})

	info, err := obj.Put(ctx, *objMeta, bytes.NewReader(data))
	if err != nil {
		return nil, wrapErr(err)
	}

	return toObjectInfo(info), nil
}

// lockObjectPut serializes PutObject calls for the same (bucket, name); the lock table is never trimmed.
func (c *Client) lockObjectPut(bucket, name string) (unlock func()) {
	key := bucket + "\x00" + name
	value, _ := c.putObjectLocks.LoadOrStore(key, &sync.Mutex{})
	mu := value.(*sync.Mutex) //nolint:errcheck // always *sync.Mutex
	mu.Lock()
	return mu.Unlock
}

// DeleteObject deletes an object from an Object Store bucket.
func (c *Client) DeleteObject(ctx context.Context, bucket string, name string) error {
	if err := validateNATSSubjectLength("object name", name); err != nil {
		return wrapErr(err)
	}

	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return wrapErr(err)
	}

	if err := obj.Delete(ctx, name); err != nil {
		return wrapErr(err)
	}

	return nil
}

// SealObjectBucket makes an Object Store bucket read-only once verifyObjectBucket accepts it.
func (c *Client) SealObjectBucket(ctx context.Context, bucket string) error {
	ctx, cancel := corecontext.ApplyTimeout(ctx, objectOperationTimeout)
	defer cancel()

	obj, err := c.jetStream.ObjectStore(ctx, bucket)
	if err != nil {
		return wrapBucketErr(err)
	}
	if err := verifyObjectBucket(ctx, obj, bucket); err != nil {
		return err
	}

	if err := obj.Seal(ctx); err != nil {
		return wrapBucketErr(err)
	}

	return nil
}

// verifyObjectBucket confirms the bucket's stream has exactly the Object Store subjects.
// It is best-effort: an unreadable status lets the caller proceed.
func verifyObjectBucket(ctx context.Context, obj jetstream.ObjectStore, bucket string) error {
	status, err := obj.Status(ctx)
	if err != nil {
		return wrapBucketErr(err)
	}
	provider, ok := status.(streamInfoProvider)
	if !ok {
		return nil
	}
	info := provider.StreamInfo()
	if info == nil {
		return nil
	}
	if objectStreamSubjectsValid(bucket, info.Config.Subjects) {
		return nil
	}
	return errors.Join(errs.ErrNotAKVOrObjectBucket, jetstream.ErrBadBucket)
}

func objectStreamSubjectsValid(bucket string, subjects []string) bool {
	if len(subjects) != 2 { //nolint:mnd // chunk + meta subjects
		return false
	}
	wantChunks := "$O." + bucket + objChunksSubjectSuffix
	wantMeta := "$O." + bucket + objMetaSubjectSuffix
	seenChunks, seenMeta := false, false
	for _, s := range subjects {
		switch s {
		case wantChunks:
			seenChunks = true
		case wantMeta:
			seenMeta = true
		}
	}
	return seenChunks && seenMeta
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

// toObjectInfo maps a jetstream object entry, including the link nested under Opts.
func toObjectInfo(info *jetstream.ObjectInfo) *entities.ObjectInfo {
	out := converter.Convert(info, &entities.ObjectInfo{})
	if info.Opts != nil && info.Opts.Link != nil {
		out.Link = &entities.ObjectLink{
			Bucket: info.Opts.Link.Bucket,
			Name:   info.Opts.Link.Name,
		}
	}
	return out
}
