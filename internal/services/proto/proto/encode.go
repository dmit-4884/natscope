// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func jsonToBinary(schema *protoutils.Schema, md protoreflect.MessageDescriptor, messageType string, data []byte) ([]byte, error) {
	msg := dynamicpb.NewMessage(md)
	if err := schema.ParseJSON(data, msg); err != nil {
		return nil, jsonConvertError(messageType, err)
	}
	out, err := protoutils.EncodeBinary(msg)
	if err != nil {
		return nil, &codecError{msg: fmt.Sprintf("Failed to encode '%s': %v", messageType, err), cause: err}
	}
	return out, nil
}

// codecError carries a user-facing message while keeping the cause for
// errors.Is checks.
type codecError struct {
	msg   string
	cause error
}

func (e *codecError) Error() string { return e.msg }

func (e *codecError) Unwrap() error { return e.cause }

// snapshotError describes a snapshot resolution failure for sourceID.
func snapshotError(sourceID string, err error) error {
	var msg string
	switch {
	case errors.Is(err, errs.ErrMappingSourceNotFound):
		msg = fmt.Sprintf("Proto source '%s' not found.", sourceID)
	case errors.Is(err, errs.ErrMappingSourceDisabled):
		msg = fmt.Sprintf("Proto source '%s' is disabled.", sourceID)
	case errors.Is(err, errs.ErrMappingSelectionMissing):
		msg = fmt.Sprintf("Proto source '%s' has no selected git ref.", sourceID)
	default:
		msg = fmt.Sprintf("Failed to resolve proto source '%s': %v", sourceID, err)
	}
	return &codecError{msg: msg, cause: err}
}

// typeNotFoundError describes a message type missing from a source snapshot.
func typeNotFoundError(messageType, sourceID string) error {
	return &codecError{
		msg:   fmt.Sprintf("Proto type '%s' not found in source '%s'.", messageType, sourceID),
		cause: errs.ErrProtoMessageNotFound,
	}
}

// jsonConvertError describes JSON that does not fit messageType.
func jsonConvertError(messageType string, err error) error {
	return &codecError{msg: fmt.Sprintf("Cannot convert JSON to '%s': %v", messageType, err), cause: err}
}

// Encode converts JSON data to protobuf binary format using the resolved
// snapshot.
func (s *Service) Encode(ctx context.Context, req entities.CodecRequest) (*entities.EncodeResult, error) {
	snap, err := s.snapshotForRequest(ctx, req)
	if err != nil {
		return &entities.EncodeResult{Success: false, Error: snapshotError(req.SourceID, err).Error()}, nil
	}

	md, ok := snap.Schema.Message(req.MessageType)
	if !ok {
		return &entities.EncodeResult{Success: false, Error: typeNotFoundError(req.MessageType, snap.SourceID).Error()}, nil
	}

	data, err := jsonToBinary(snap.Schema, md, req.MessageType, req.JSON)
	if err != nil {
		return &entities.EncodeResult{Success: false, Error: err.Error()}, nil
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
		return nil, snapshotError(req.SourceID, err)
	}
	md, ok := snap.Schema.Message(req.MessageType)
	if !ok {
		return nil, typeNotFoundError(req.MessageType, snap.SourceID)
	}
	return jsonToBinary(snap.Schema, md, req.MessageType, req.JSON)
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
		return &entities.ValidationResult{Valid: false, Error: snapshotError(req.SourceID, err).Error()}, nil
	}
	md, ok := snap.Schema.Message(req.MessageType)
	if !ok {
		return &entities.ValidationResult{Valid: false, Error: typeNotFoundError(req.MessageType, snap.SourceID).Error()}, nil
	}
	return snap.Schema.Validate(md, data), nil
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
