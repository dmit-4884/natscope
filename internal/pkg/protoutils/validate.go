// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package protoutils

import (
	"errors"
	"fmt"
	"slices"

	"buf.build/go/protovalidate"

	"github.com/dmit-4884/natscope/internal/entities"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	atlasslices "github.com/altessa-s/go-atlas/core/collections/slices"
)

// Validate checks wire bytes of type md against its buf.validate rules.
func (s *Schema) Validate(md protoreflect.MessageDescriptor, data []byte) *entities.ValidationResult {
	pv, err := protovalidate.New(
		protovalidate.WithMessageDescriptors(md),
		protovalidate.WithExtensionTypeResolver(extensionResolver{s.Types}),
	)
	if err != nil {
		return &entities.ValidationResult{Error: fmt.Sprintf("Failed to create validator: %v", err)}
	}

	msg := dynamicpb.NewMessage(md)
	if decodeErr := s.ParseBinary(data, msg); decodeErr != nil {
		return &entities.ValidationResult{Error: fmt.Sprintf("Cannot decode message: %v", decodeErr)}
	}

	err = pv.Validate(msg)
	if err == nil {
		return &entities.ValidationResult{Valid: true, Violations: []*entities.ValidationViolation{}}
	}
	if ve, ok := errors.AsType[*protovalidate.ValidationError](err); ok {
		return &entities.ValidationResult{
			Violations: slices.Collect(atlasslices.Map(ve.Violations, func(v *protovalidate.Violation) *entities.ValidationViolation {
				return &entities.ValidationViolation{
					FieldPath:    protovalidate.FieldPathString(v.Proto.GetField()),
					Message:      v.Proto.GetMessage(),
					ConstraintId: v.Proto.GetRuleId(),
				}
			})),
		}
	}
	if ce, ok := errors.AsType[*protovalidate.CompilationError](err); ok {
		return &entities.ValidationResult{Error: fmt.Sprintf("Validation rules do not compile: %v", ce)}
	}
	if re, ok := errors.AsType[*protovalidate.RuntimeError](err); ok {
		return &entities.ValidationResult{Error: fmt.Sprintf("Validation rule failed at runtime: %v", re)}
	}
	return &entities.ValidationResult{Error: fmt.Sprintf("Validation failed: %v", err)}
}
