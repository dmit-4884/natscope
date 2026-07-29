// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"buf.build/go/protovalidate"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	atlasslices "github.com/altessa-s/go-atlas/core/collections/slices"
)

// celPatterns are error patterns indicating CEL/extension validation issues.
var celPatterns = []string{"unknown rules", "extensiontyperesolver", "cel", "unknown extension", "compile error"}

// Validate validates raw protobuf bytes against buf.validate rules.
func Validate(md protoreflect.MessageDescriptor, data []byte) (*entities.ValidationResult, error) {
	// Build a custom extension resolver
	extResolver := newExtResolver(md)

	// Create validator for this message type
	pv, err := protovalidate.New(
		protovalidate.WithMessageDescriptors(md),
		protovalidate.WithExtensionTypeResolver(extResolver),
	)
	if err != nil {
		return &entities.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Failed to create validator: %v", err),
		}, nil
	}

	// Create dynamic message and unmarshal
	msg := dynamicpb.NewMessage(md)
	if err = proto.Unmarshal(data, msg); err != nil {
		return &entities.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("Cannot decode message: %v", err),
		}, nil
	}

	// Validate
	err = pv.Validate(msg)
	if err == nil {
		return &entities.ValidationResult{
			Valid:      true,
			Violations: []*entities.ValidationViolation{},
		}, nil
	}

	// Check if it's a validation error
	if ve, ok := errors.AsType[*protovalidate.ValidationError](err); ok {
		violations := slices.Collect(atlasslices.Map(
			ve.Violations,
			func(v *protovalidate.Violation) *entities.ValidationViolation {
				return &entities.ValidationViolation{
					FieldPath:    protovalidate.FieldPathString(v.Proto.GetField()),
					Message:      v.Proto.GetMessage(),
					ConstraintId: v.Proto.GetRuleId(),
				}
			},
		))
		return &entities.ValidationResult{
			Valid:      false,
			Violations: violations,
		}, nil
	}

	// Check if it's a CEL/extension error
	errStr := err.Error()
	errStrLower := strings.ToLower(errStr)

	if atlasslices.Any(celPatterns, func(pattern string) bool {
		return strings.Contains(errStrLower, pattern)
	}) {
		return &entities.ValidationResult{
			Valid: false,
			Error: fmt.Sprintf("CEL validation error: %s", errStr),
		}, nil
	}

	return &entities.ValidationResult{
		Valid: false,
		Error: fmt.Sprintf("Validation failed: %v", err),
	}, nil
}
