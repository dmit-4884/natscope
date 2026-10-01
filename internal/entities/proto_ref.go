// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// RefKind is the kind of a git ref.
type RefKind string

const (
	RefKindTag    RefKind = "tag"
	RefKindBranch RefKind = "branch"
	RefKindCommit RefKind = "commit"
)

// ProtoRef is a git ref and the commit it points to.
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
