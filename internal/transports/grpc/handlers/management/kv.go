// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package management

import (
	"context"
	"encoding/base64"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"

	managementpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/nats/v1/management"
	natspb "github.com/dmit-4884/natscope/proto/gen/types/nats"
)

// ListKVBuckets lists all KeyValue buckets.
func (h *Handler) ListKVBuckets(
	ctx context.Context,
	req *connect.Request[managementpb.ListKVBucketsRequest],
) (*connect.Response[managementpb.ListKVBucketsResponse], error) {
	buckets, err := h.natsService.ListKVBuckets(ctx, req.Msg.GetConnectionId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListKVBucketsResponse{
		Buckets: slices.To(buckets, func(b entities.KVBucketInfo) *natspb.KVBucketInfo {
			return converter.Convert(&b, &natspb.KVBucketInfo{}, replicasMapping, protoCodecs)
		}),
	}), nil
}

// CreateKVBucket creates a new KeyValue bucket.
func (h *Handler) CreateKVBucket(
	ctx context.Context,
	req *connect.Request[managementpb.CreateKVBucketRequest],
) (*connect.Response[managementpb.CreateKVBucketResponse], error) {
	in := req.Msg
	cfg := in.GetConfig()

	cr := converter.Convert(cfg, &entities.KVBucketConfig{},
		protoCodecs,
		converter.WithFieldMappings(map[string]string{"NumReplicas": "Replicas"}),
		converter.WithIgnoreFields("Placement", "History", "Mirror", "Sources", "Republish"),
	)
	cr.History = uint8(cfg.GetHistory())

	if cfg.GetPlacement() != nil {
		cr.Placement = converter.Convert(cfg.GetPlacement(), &entities.Placement{})
	}
	if cfg.GetMirror() != nil {
		cr.Mirror = streamSourceRefToEntity(cfg.GetMirror())
	}
	cr.Sources = slices.To(cfg.GetSources(), streamSourceRefToEntity)
	if cfg.GetRepublish() != nil {
		cr.Republish = converter.Convert(cfg.GetRepublish(), &entities.StreamRePublish{},
			converter.WithFieldMappings(map[string]string{"Source": "Src", "Destination": "Dest"}),
		)
	}

	bucket, err := h.natsService.CreateKVBucket(ctx, in.GetConnectionId(), *cr)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.CreateKVBucketResponse{
		Bucket: converter.Convert(bucket, &natspb.KVBucketInfo{}, replicasMapping, protoCodecs),
	}), nil
}

// GetKVBucket gets information about a KeyValue bucket.
func (h *Handler) GetKVBucket(
	ctx context.Context,
	req *connect.Request[managementpb.GetKVBucketRequest],
) (*connect.Response[managementpb.GetKVBucketResponse], error) {
	in := req.Msg
	info, err := h.natsService.GetKVBucket(ctx, in.GetConnectionId(), in.GetBucket())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.GetKVBucketResponse{
		Bucket: converter.Convert(info, &natspb.KVBucketInfo{}, replicasMapping, protoCodecs),
	}), nil
}

// DeleteKVBucket deletes a KeyValue bucket.
func (h *Handler) DeleteKVBucket(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteKVBucketRequest],
) (*connect.Response[managementpb.DeleteKVBucketResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteKVBucket(ctx, in.GetConnectionId(), in.GetBucket()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteKVBucketResponse{}), nil
}

// ListKVKeys lists all keys in a KeyValue bucket.
func (h *Handler) ListKVKeys(
	ctx context.Context,
	req *connect.Request[managementpb.ListKVKeysRequest],
) (*connect.Response[managementpb.ListKVKeysResponse], error) {
	in := req.Msg
	keys, err := h.natsService.ListKVKeys(ctx, in.GetConnectionId(), in.GetBucket())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListKVKeysResponse{Keys: keys}), nil
}

// GetKVKey gets a key from a KeyValue bucket.
func (h *Handler) GetKVKey(
	ctx context.Context,
	req *connect.Request[managementpb.GetKVKeyRequest],
) (*connect.Response[managementpb.GetKVKeyResponse], error) {
	in := req.Msg
	entry, err := h.natsService.GetKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.GetKVKeyResponse{
		Entry: converter.Convert(entry, &natspb.KVEntry{}, protoCodecs),
	}), nil
}

// GetKVKeyHistory returns the revision history for a key, oldest first.
func (h *Handler) GetKVKeyHistory(
	ctx context.Context,
	req *connect.Request[managementpb.GetKVKeyHistoryRequest],
) (*connect.Response[managementpb.GetKVKeyHistoryResponse], error) {
	in := req.Msg
	entries, err := h.natsService.GetKVKeyHistory(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.GetKVKeyHistoryResponse{
		Entries: slices.To(entries, func(e entities.KVEntry) *natspb.KVEntry {
			return converter.Convert(&e, &natspb.KVEntry{}, protoCodecs)
		}),
	}), nil
}

// PutKVKey puts a key in a KeyValue bucket.
func (h *Handler) PutKVKey(
	ctx context.Context,
	req *connect.Request[managementpb.PutKVKeyRequest],
) (*connect.Response[managementpb.PutKVKeyResponse], error) {
	in := req.Msg
	value, err := base64.StdEncoding.DecodeString(in.GetValue())
	if err != nil {
		// Treat as plain text if not valid base64.
		value = []byte(in.GetValue())
	}

	revision, err := h.natsService.PutKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey(), value, in.GetRevision())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PutKVKeyResponse{Revision: revision}), nil
}

// DeleteKVKey deletes a key from a KeyValue bucket.
func (h *Handler) DeleteKVKey(
	ctx context.Context,
	req *connect.Request[managementpb.DeleteKVKeyRequest],
) (*connect.Response[managementpb.DeleteKVKeyResponse], error) {
	in := req.Msg
	if err := h.natsService.DeleteKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.DeleteKVKeyResponse{}), nil
}

// PurgeKVKey purges all revisions of a key from a KeyValue bucket.
func (h *Handler) PurgeKVKey(
	ctx context.Context,
	req *connect.Request[managementpb.PurgeKVKeyRequest],
) (*connect.Response[managementpb.PurgeKVKeyResponse], error) {
	in := req.Msg
	if err := h.natsService.PurgeKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PurgeKVKeyResponse{}), nil
}
