// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

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

	// ErrMappingSelectionMissing is returned when no selection exists for a git source mapping points to.
	ErrMappingSelectionMissing = errors.New("mapping: selection missing for source")

	// ErrMappingDescriptorMissing is returned when no compiled descriptor exists for the resolved (source, tag).
	ErrMappingDescriptorMissing = errors.New("mapping: descriptor missing for source/tag")

	// ErrMessageTypeNotInSource is returned when a message type is not present in the active snapshot of the mapping's source.
	ErrMessageTypeNotInSource = errors.New("mapping: message type not in source snapshot")
)
