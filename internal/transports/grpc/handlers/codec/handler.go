// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package codec

import (
	"context"
	"encoding/base64"
	"net/http"

	"connectrpc.com/connect"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"
	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/transports/grpc/helpers"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	codecpb "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec"
	codecconnect "github.com/dmit-4884/natscope/proto/gen/services/grpc/proto/v1/codec/grpc_proto_codecconnect"
	protopb "github.com/dmit-4884/natscope/proto/gen/types/proto"
)

// Handler implements the Connect CodecServiceHandler interface.
type Handler struct {
	protoService protosvc.Codec
}

// New creates a new CodecService handler.
func New(protoService protosvc.Codec) *Handler {
	return &Handler{protoService: protoService}
}

// HTTPHandler returns the Connect route + handler; proto codec errors use the
// shared interceptor.
func (h *Handler) HTTPHandler(opts ...connect.HandlerOption) (string, http.Handler) {
	opts = append(opts, connect.WithInterceptors(
		grpchelpers.NewHandlerErrorInterceptor(nil),
	))
	return codecconnect.NewCodecServiceHandler(h, opts...)
}

// DecodeMessage decodes protobuf binary data to JSON within the resolved
// snapshot.
func (h *Handler) DecodeMessage(
	ctx context.Context,
	req *connect.Request[codecpb.DecodeMessageRequest],
) (*connect.Response[codecpb.DecodeMessageResponse], error) {
	in := req.Msg
	result, err := h.protoService.Decode(ctx, entities.CodecRequest{
		Data:        in.Data,
		SourceID:    in.SourceId,
		Fingerprint: ptr.Unwrap(in.Fingerprint, ""),
		MessageType: in.MessageType,
		Framing:     grpchelpers.FramingFromProto(in.Framing),
	})
	if err != nil {
		return nil, err
	}

	pbResult := &protopb.DecodeResult{
		Data:        string(result.Decoded),
		MessageType: result.MessageType,
		ValidBytes:  int32(result.ValidBytes), //nolint:gosec // bounded by the request size
		UnknownFields: slices.To(result.UnknownFields, func(f entities.UnknownField) *protopb.UnknownField {
			pb := converter.Convert(f, &protopb.UnknownField{}, converter.WithIgnoreFields("WireType"))
			pb.WireType = wireTypes[f.WireType]
			return pb
		}),
	}
	if result.Error != "" {
		pbResult.Error = &result.Error
	}
	return connect.NewResponse(&codecpb.DecodeMessageResponse{Result: pbResult}), nil
}

// DecodeWire reads protobuf binary data without a schema.
func (h *Handler) DecodeWire(
	_ context.Context,
	req *connect.Request[codecpb.DecodeWireRequest],
) (*connect.Response[codecpb.DecodeWireResponse], error) {
	dump := h.protoService.DecodeWire(req.Msg.Data)
	resp := &codecpb.DecodeWireResponse{
		Fields:     wireFieldsToProto(dump.Fields),
		ValidBytes: int32(dump.ValidBytes), //nolint:gosec // bounded by the request size
	}
	if dump.Error != "" {
		resp.Error = &dump.Error
	}
	return connect.NewResponse(resp), nil
}

var wireTypes = map[entities.WireType]protopb.WireType{
	entities.WireVarint:  protopb.WireType_WIRE_TYPE_VARINT,
	entities.WireFixed64: protopb.WireType_WIRE_TYPE_FIXED64,
	entities.WireBytes:   protopb.WireType_WIRE_TYPE_BYTES,
	entities.WireGroup:   protopb.WireType_WIRE_TYPE_GROUP,
	entities.WireFixed32: protopb.WireType_WIRE_TYPE_FIXED32,
}

func wireFieldsToProto(fields []*entities.WireField) []*protopb.WireField {
	return slices.To(fields, func(f *entities.WireField) *protopb.WireField {
		pb := converter.Convert(f, &protopb.WireField{}, converter.WithIgnoreFields("WireType", "Message"))
		pb.WireType = wireTypes[f.WireType]
		pb.Message = wireFieldsToProto(f.Message)
		return pb
	})
}

// EncodeMessage encodes JSON data to protobuf binary within the resolved
// snapshot.
func (h *Handler) EncodeMessage(
	ctx context.Context,
	req *connect.Request[codecpb.EncodeMessageRequest],
) (*connect.Response[codecpb.EncodeMessageResponse], error) {
	in := req.Msg
	result, err := h.protoService.Encode(ctx, entities.CodecRequest{
		JSON:        []byte(in.Data),
		SourceID:    in.SourceId,
		Fingerprint: ptr.Unwrap(in.Fingerprint, ""),
		MessageType: in.MessageType,
		Framing:     grpchelpers.FramingFromProto(in.Framing),
	})
	if err != nil {
		return nil, err
	}

	pbResult := &protopb.EncodeResult{DataSize: int64(result.DataSize)}
	if result.DataBase64 != "" {
		data, decErr := base64.StdEncoding.DecodeString(result.DataBase64)
		if decErr == nil {
			pbResult.Data = data
		}
	}
	if result.Error != "" {
		pbResult.Error = &result.Error
	}
	return connect.NewResponse(&codecpb.EncodeMessageResponse{Result: pbResult}), nil
}

// ValidateMessage validates protobuf binary data against validation rules
// within the resolved snapshot.
func (h *Handler) ValidateMessage(
	ctx context.Context,
	req *connect.Request[codecpb.ValidateMessageRequest],
) (*connect.Response[codecpb.ValidateMessageResponse], error) {
	in := req.Msg
	dataBase64 := base64.StdEncoding.EncodeToString(in.Data)
	result, err := h.protoService.Validate(ctx, dataBase64, entities.CodecRequest{
		SourceID:    in.SourceId,
		Fingerprint: ptr.Unwrap(in.Fingerprint, ""),
		MessageType: in.MessageType,
	})
	if err != nil {
		return nil, err
	}

	pbResult := &protopb.ValidationResult{Valid: result.Valid}
	if result.Error != "" {
		pbResult.Error = &result.Error
	}
	pbResult.Violations = slices.To(
		result.Violations,
		func(v *entities.ValidationViolation) *protopb.ValidationViolation {
			return converter.Convert(v, &protopb.ValidationViolation{})
		},
	)
	return connect.NewResponse(&codecpb.ValidateMessageResponse{Result: pbResult}), nil
}
