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
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

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
		Buckets: *converter.Convert(buckets, &[]*natspb.KVBucketInfo{}, replicasMapping, protoCodecs),
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
		replicasMappingToEntity,
		converter.WithIgnoreFields("History"),
	)
	cr.History = uint8(cfg.GetHistory())

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

// UpdateKVBucket applies new settings to a KeyValue bucket.
func (h *Handler) UpdateKVBucket(
	ctx context.Context,
	req *connect.Request[managementpb.UpdateKVBucketRequest],
) (*connect.Response[managementpb.UpdateKVBucketResponse], error) {
	in := req.Msg
	settings := converter.Convert(in.GetSettings(), &entities.KVBucketSettings{},
		protoCodecs,
		replicasMappingToEntity,
		converter.WithIgnoreFields("History"),
	)
	settings.History = uint8(in.GetSettings().GetHistory())

	bucket, err := h.natsService.UpdateKVBucket(ctx, in.GetConnectionId(), in.GetBucket(), *settings)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.UpdateKVBucketResponse{
		Bucket: converter.Convert(bucket, &natspb.KVBucketInfo{}, replicasMapping, protoCodecs),
	}), nil
}

// PurgeKVBucket removes every key and revision of a KeyValue bucket.
func (h *Handler) PurgeKVBucket(
	ctx context.Context,
	req *connect.Request[managementpb.PurgeKVBucketRequest],
) (*connect.Response[managementpb.PurgeKVBucketResponse], error) {
	in := req.Msg
	if err := h.natsService.PurgeKVBucket(ctx, in.GetConnectionId(), in.GetBucket()); err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PurgeKVBucketResponse{}), nil
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
	list, err := h.natsService.ListKVKeys(ctx, in.GetConnectionId(), in.GetBucket(), entities.KVKeysQuery{
		Filter: in.GetFilter(),
		Limit:  int(in.GetLimit()),
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.ListKVKeysResponse{Keys: list.Keys, Truncated: list.Truncated}), nil
}

// GetKVKey gets a key from a KeyValue bucket, its value decoded when a mapping or detected type applies.
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
		Entry: h.kvEntryToProto(ctx, entry, h.detectsTypes(ctx)),
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
	detect := h.detectsTypes(ctx)
	return connect.NewResponse(&managementpb.GetKVKeyHistoryResponse{
		Entries: slices.To(entries, func(e entities.KVEntry) *natspb.KVEntry {
			return h.kvEntryToProto(ctx, &e, detect)
		}),
	}), nil
}

func (h *Handler) kvEntryToProto(ctx context.Context, e *entities.KVEntry, detect bool) *natspb.KVEntry {
	pb := converter.Convert(e, &natspb.KVEntry{}, protoCodecs)
	data, err := base64.StdEncoding.DecodeString(e.Value)
	if err != nil || len(data) == 0 {
		return pb
	}
	pb.Decoded = grpchelpers.DecodeResultToProto(h.codec.DecodeSubject(ctx, e.Subject(), data, detect))
	return pb
}

func (h *Handler) detectsTypes(ctx context.Context) bool {
	cfg, err := h.settings.Get(ctx)
	return err != nil || cfg.DetectsTypes()
}

// PutKVKey stores base64 bytes, or JSON encoded as a Protobuf message, under a key.
func (h *Handler) PutKVKey(
	ctx context.Context,
	req *connect.Request[managementpb.PutKVKeyRequest],
) (*connect.Response[managementpb.PutKVKeyResponse], error) {
	in := req.Msg
	value, err := h.kvValue(ctx, in)
	if err != nil {
		return nil, err
	}

	var revision uint64
	if ttl := in.GetTtl().AsDuration(); ttl > 0 {
		revision, err = h.natsService.CreateKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey(), value, ttl)
	} else {
		revision, err = h.natsService.PutKVKey(ctx, in.GetConnectionId(), in.GetBucket(), in.GetKey(), value, in.GetRevision())
	}
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&managementpb.PutKVKeyResponse{Revision: revision}), nil
}

func (h *Handler) kvValue(ctx context.Context, in *managementpb.PutKVKeyRequest) ([]byte, error) {
	pv := in.GetProto()
	if pv == nil {
		value, err := base64.StdEncoding.DecodeString(in.GetValue())
		if err != nil {
			return nil, &errs.NATSValidationError{Description: "value must be base64-encoded", Cause: err}
		}
		return value, nil
	}
	value, err := h.codec.EncodeRaw(ctx, entities.CodecRequest{
		JSON:        []byte(pv.GetJson()),
		SourceID:    pv.GetSourceId(),
		MessageType: pv.GetMessageType(),
		Fingerprint: pv.GetFingerprint(),
		Framing:     grpchelpers.FramingFromProto(pv.GetFraming()),
	})
	if err != nil {
		return nil, &errs.ProtoEncodeError{Description: err.Error()}
	}
	return value, nil
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
