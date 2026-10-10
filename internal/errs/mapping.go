// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package errs

import "errors"

var (
	// ErrMappingNotFound is returned when a subject mapping cannot be found.
	ErrMappingNotFound = errors.New("mapping: not found")

	// ErrMappingPatternAlreadyInUse is returned when (pattern, source_id) is already mapped.
	ErrMappingPatternAlreadyInUse = errors.New("mapping: pattern already in use for this source")

	// ErrMappingSourceIDRequired is returned when a mapping is created/updated without a source_id.
	ErrMappingSourceIDRequired = errors.New("mapping: source_id is required")

	// ErrMappingSourceNotFound is returned when the source referenced by a mapping no longer exists.
	ErrMappingSourceNotFound = errors.New("mapping: source not found")

	// ErrMappingSourceDisabled is returned when the source referenced by a mapping is disabled.
	ErrMappingSourceDisabled = errors.New("mapping: source is disabled")

	// ErrMappingSelectionMissing is returned when the mapping's source has no active schema yet.
	ErrMappingSelectionMissing = errors.New("mapping: selection missing for source")

	// ErrMappingDescriptorMissing is returned when the schema a mapping resolves to is not stored.
	ErrMappingDescriptorMissing = errors.New("mapping: compiled schema missing")

	// ErrMessageTypeNotInSource is returned when a message type is not present in the active snapshot of the mapping's source.
	ErrMessageTypeNotInSource = errors.New("mapping: message type not in source snapshot")

	// ErrMappingPatternRequired is returned when pattern is empty after trimming.
	ErrMappingPatternRequired = errors.New("mapping: pattern is required")

	// ErrMappingMessageTypeRequired is returned when message_type is empty after trimming.
	ErrMappingMessageTypeRequired = errors.New("mapping: message_type is required")

	// ErrMappingDuplicateInBatch is returned when a batch repeats a (pattern, source_id) key.
	ErrMappingDuplicateInBatch = errors.New("mapping: duplicate (pattern, source_id) in batch")
)
