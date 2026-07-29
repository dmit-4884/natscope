// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// ListObjectBuckets lists all Object Store buckets.
func (h *Handler) ListObjectBuckets(
	ctx context.Context,
	req *connect.Request[managementpb.ListObjectBucketsRequest],
) (*connect.Response[managementpb.ListObjectBucketsResponse], error) {
	buckets, err := h.natsService.ListObjectBuckets(ctx, req.Msg.GetConnectionId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListObjectBucketsResponse{
		Buckets: slices.To(buckets, func(b entities.ObjectBucketInfo) *natspb.ObjectBucketInfo {
			return converter.Convert(&b, &natspb.ObjectBucketInfo{}, replicasMapping)
		}),
	}), nil
}

// CreateObjectBucket creates a new Object Store bucket.
func (h *Handler) CreateObjectBucket(
	ctx context.Context,
	req *connect.Request[managementpb.CreateObjectBucketRequest],
) (*connect.Response[managementpb.CreateObjectBucketResponse], error) {
	in := req.Msg
	cfg := in.GetConfig()

	cr := converter.Convert(cfg, &entities.ObjectBucketConfig{},
		protoCodecs,
		converter.WithFieldMappings(map[string]string{"NumReplicas": "Replicas"}),
		converter.WithIgnoreFields("Placement"),
	)

	if cfg.GetPlacement() != nil {
		cr.Placement = converter.Convert(cfg.GetPlacement(), &entities.Placement{})
	}

	bucket, err := h.natsService.CreateObjectBucket(ctx, in.GetConnectionId(), *cr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.CreateObjectBucketResponse{
		Bucket: converter.Convert(bucket, &natspb.ObjectBucketInfo{}, replicasMapping),
	}), nil
}

// GetObjectBucket gets information about an Object Store bucket.
func (h *Handler) GetObjectBucket(
	ctx context.Context,
	req *connect.Request[managementpb.GetObjectBucketRequest],
) (*connect.Response[managementpb.GetObjectBucketResponse], error) {
	in := req.Msg
	info, err := h.natsService.GetObjectBucket(ctx, in.GetConnectionId(), in.GetBucket())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.GetObjectBucketResponse{
		Bucket: converter.Convert(info, &natspb.ObjectBucketInfo{}, replicasMapping),
	}), nil
}

// DeleteObjectBucket deletes an Object Store bucket.
func (h *Handler) DeleteObjectBucket(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteObjectBucketRequest],
) (*connect.Response[managementpb.DeleteObjectBucketResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteObjectBucket(ctx, in.GetConnectionId(), in.GetBucket()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteObjectBucketResponse{}), nil
}

// SealObjectBucket seals an Object Store bucket (makes it read-only).
func (h *Handler) SealObjectBucket(
	ctx context.Context,
	req *connect.Request[managementpb.SealObjectBucketRequest],
) (*connect.Response[managementpb.SealObjectBucketResponse], error) {
	in := req.Msg
	if err := h.natsService.SealObjectBucket(ctx, in.GetConnectionId(), in.GetBucket()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.SealObjectBucketResponse{}), nil
}

// ListObjects lists all objects in an Object Store bucket.
func (h *Handler) ListObjects(
	ctx context.Context,
	req *connect.Request[managementpb.ListObjectsRequest],
) (*connect.Response[managementpb.ListObjectsResponse], error) {
	in := req.Msg
	objects, err := h.natsService.ListObjects(ctx, in.GetConnectionId(), in.GetBucket())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListObjectsResponse{
		Objects: slices.To(objects, func(o *entities.ObjectInfo) *natspb.ObjectInfo {
			return converter.Convert(o, &natspb.ObjectInfo{}, protoCodecs)
		}),
	}), nil
}

// GetObject gets an object from an Object Store bucket.
func (h *Handler) GetObject(
	ctx context.Context,
	req *connect.Request[managementpb.GetObjectRequest],
) (*connect.Response[managementpb.GetObjectResponse], error) {
	in := req.Msg
	data, info, err := h.natsService.GetObject(ctx, in.GetConnectionId(), in.GetBucket(), in.GetName())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.GetObjectResponse{
		Info: converter.Convert(info, &natspb.ObjectInfo{}, protoCodecs),
		Data: data,
	}), nil
}

// PutObject stores an object in an Object Store bucket.
func (h *Handler) PutObject(
	ctx context.Context,
	req *connect.Request[managementpb.PutObjectRequest],
) (*connect.Response[managementpb.PutObjectResponse], error) {
	in := req.Msg
	meta := *converter.Convert(in, &entities.ObjectMeta{})
	info, err := h.natsService.PutObject(ctx, in.GetConnectionId(), in.GetBucket(), meta, in.GetData())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PutObjectResponse{
		Info: converter.Convert(info, &natspb.ObjectInfo{}, protoCodecs),
	}), nil
}

// DeleteObject deletes an object from an Object Store bucket.
func (h *Handler) DeleteObject(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteObjectRequest],
) (*connect.Response[managementpb.DeleteObjectResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteObject(ctx, in.GetConnectionId(), in.GetBucket(), in.GetName()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteObjectResponse{}), nil
}
