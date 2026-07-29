// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ConflictKind classifies a schema conflict detected during snapshot
// aggregation.
type ConflictKind string

const (
	// DuplicateFileDifferentContent same proto file path but content differs
	// (versioning bug or accidental fork).
	DuplicateFileDifferentContent ConflictKind = "duplicate_file_different_content"

	// SameSymbolSameShape same FQN in two files with identical wire-shape.
	SameSymbolSameShape ConflictKind = "same_symbol_same_shape"

	// SameSymbolDifferentShape same FQN, different wire shape; decoding the wrong
	// one silently mis-interprets bytes.
	SameSymbolDifferentShape ConflictKind = "same_symbol_different_shape"
)

// ConflictSeverity drives whether the policy gates compilation.
type ConflictSeverity string

const (
	SeverityInfo  ConflictSeverity = "info"
	SeverityError ConflictSeverity = "error"
)

// SchemaRef points to a specific file in a specific snapshot.
type SchemaRef struct {
	SourceID string
	Tag      string
	File     string
}

// SchemaConflict records one merge conflict; persisted so the UI lists
// conflicts without re-running merge.
type SchemaConflict struct {
	BaseEntity

	Kind     ConflictKind
	Severity ConflictSeverity

	// Symbol identifies the conflict: file name for file-kinds, FQN for
	// symbol-kinds.
	Symbol string

	Winner SchemaRef
	Loser  SchemaRef

	Reason string
	Policy string
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
