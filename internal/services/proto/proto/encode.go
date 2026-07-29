// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Encode converts JSON data to protobuf binary format using the resolved
// snapshot.
func (s *Service) Encode(ctx context.Context, req entities.CodecRequest) (*entities.EncodeResult, error) {
	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return &entities.EncodeResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to resolve snapshot: %v", err),
		}, nil
	}

	md, ok := snap.Messages[req.MessageType]
	if !ok {
		return &entities.EncodeResult{
			Success: false,
			Error:   fmt.Sprintf("Proto type '%s' not found in source '%s'.", req.MessageType, snap.SourceID),
		}, nil
	}

	msg := dynamicpb.NewMessage(md)
	if unmarshalErr := (protojson.UnmarshalOptions{
		DiscardUnknown: true,
	}).Unmarshal(req.JSON, msg); unmarshalErr != nil {
		return &entities.EncodeResult{
			Success: false,
			Error:   fmt.Sprintf("Cannot convert JSON to '%s': %v", req.MessageType, unmarshalErr),
		}, nil
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return &entities.EncodeResult{
			Success: false,
			Error:   fmt.Sprintf("Failed to encode: %v", err),
		}, nil
	}

	return &entities.EncodeResult{
		Success:    true,
		DataBase64: base64.StdEncoding.EncodeToString(data),
		DataSize:   len(data),
	}, nil
}

// EncodeRaw converts JSON data to raw protobuf bytes using the resolved
// snapshot.
func (s *Service) EncodeRaw(ctx context.Context, req entities.CodecRequest) ([]byte, error) {
	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	md, ok := snap.Messages[req.MessageType]
	if !ok {
		return nil, errs.ErrProtoMessageNotFound
	}

	msg := dynamicpb.NewMessage(md)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(req.JSON, msg); err != nil {
		return nil, errors.Wrapf(err, "cannot convert JSON to '%s'", req.MessageType)
	}
	return proto.Marshal(msg)
}

// Validate validates protobuf data against buf.validate rules within the
// resolved snapshot.
func (s *Service) Validate(
	ctx context.Context,
	dataBase64 string,
	req entities.CodecRequest,
) (*entities.ValidationResult, error) {
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return &entities.ValidationResult{Valid: false, Error: "Invalid base64 data."}, nil
	}

	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return &entities.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to resolve snapshot: %v", err),
		}, nil
	}
	md, ok := snap.Messages[req.MessageType]
	if !ok {
		return &entities.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Proto type '%s' not found in source '%s'.", req.MessageType, snap.SourceID),
		}, nil
	}
	return protoutils.Validate(md, data)
}

// EncodeWithValidation encodes JSON to protobuf and validates the result.
func (s *Service) EncodeWithValidation(
	ctx context.Context,
	req entities.CodecRequest,
) (*entities.EncodeResult, []*entities.ValidationViolation, error) {
	result, err := s.Encode(ctx, req)
	if err != nil {
		return result, nil, err
	}
	if !result.Success {
		return result, nil, nil
	}

	validationResult, err := s.Validate(ctx, result.DataBase64, req)
	if err != nil {
		return result, nil, err
	}
	return result, validationResult.Violations, nil
}

// ValidateJSON is the one-shot encode+validate pipeline; encode, snapshot,
// and message-type failures surface as Valid:false, not error.
func (s *Service) ValidateJSON(ctx context.Context, req entities.CodecRequest) (*entities.ValidationResult, error) {
	result, violations, err := s.EncodeWithValidation(ctx, req)
	if err != nil {
		msg := "Failed to encode/validate message: " + err.Error()
		return &entities.ValidationResult{Valid: false, Error: msg}, nil
	}
	if !result.Success {
		return &entities.ValidationResult{Valid: false, Error: result.Error}, nil
	}
	return &entities.ValidationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
	}, nil
}
