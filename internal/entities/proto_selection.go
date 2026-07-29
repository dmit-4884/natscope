// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoSelection links a source to its selected tag.
type ProtoSelection struct {
	BaseEntity

	SourceID string

	// Tag is the selected Git tag (e.g., "v0.72.0").
	Tag string
}

// ProtoSelectionNew creates a new ProtoSelection with generated Id and
// timestamps.
func ProtoSelectionNew(init ...func(*ProtoSelection)) *ProtoSelection {
	s := &ProtoSelection{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](s)
	}

	return s
}

// ProtoSelections is a slice of ProtoSelection pointers.
type ProtoSelections []*ProtoSelection

// ProtoSelectionsList is the listing criteria for selections.
type ProtoSelectionsList struct {
	ListBase
	SourceID *string
}

// ProtoSelectionCreate is the create/update DTO for a selection.
type ProtoSelectionCreate struct {
	SourceID string `normalize:"trim"`
	Tag      string `normalize:"trim"`
}
