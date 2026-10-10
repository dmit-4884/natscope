// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// RefKind is the kind of a Git or BSR ref.
type RefKind string

const (
	RefKindTag    RefKind = "tag"
	RefKindBranch RefKind = "branch"
	RefKindCommit RefKind = "commit"
	RefKindLabel  RefKind = "label"
)

// Movable reports whether the ref can point at another commit later.
func (k RefKind) Movable() bool {
	return k == RefKindBranch || k == RefKindLabel
}

// ProtoRef is a Git or BSR ref and the commit it points to.
type ProtoRef struct {
	Name     string
	Kind     RefKind
	Revision string
}

// SchemaRevision describes a stored compiled schema of a source.
type SchemaRevision struct {
	Revision     string
	Fingerprint  string
	CompiledAt   int64
	MessageCount int32
	Active       bool
}
