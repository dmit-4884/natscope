// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// ConflictKind classifies a clash between two enabled proto sources.
type ConflictKind string

const (
	// ConflictFileContent is one file path with different content.
	ConflictFileContent ConflictKind = "file_content"

	// ConflictSameShape is one type name defined identically twice.
	ConflictSameShape ConflictKind = "same_shape"

	// ConflictDifferentShape is one type name defined two different ways.
	ConflictDifferentShape ConflictKind = "different_shape"
)

// ConflictSeverity tells a harmless duplicate from a real clash.
type ConflictSeverity string

const (
	SeverityInfo  ConflictSeverity = "info"
	SeverityError ConflictSeverity = "error"
)

// SchemaRef points to a file in one revision of a proto source.
type SchemaRef struct {
	SourceID string
	Revision string
	File     string
}

// SchemaConflict is a clash between two enabled proto sources, recomputed on every schema reload.
type SchemaConflict struct {
	BaseEntity

	Kind     ConflictKind
	Severity ConflictSeverity

	// Symbol is the file path of a file conflict, the fully-qualified type name otherwise.
	Symbol string

	First  SchemaRef
	Second SchemaRef

	Reason string
}

// SchemaConflictNew creates a new SchemaConflict with generated Id and
// timestamps.
func SchemaConflictNew(init ...func(*SchemaConflict)) *SchemaConflict {
	c := &SchemaConflict{BaseEntity: *New()}
	if len(init) > 0 && init[0] != nil {
		init[0](c)
	}
	return c
}

// SchemaConflicts is a slice alias for the generic List/Paginate helpers.
type SchemaConflicts []*SchemaConflict

// SchemaStatus is the loaded schema at a glance.
type SchemaStatus struct {
	// MessageTypes counts the message types of every enabled source.
	MessageTypes int
	Conflicts    SchemaConflicts
}
