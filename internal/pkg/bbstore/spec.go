// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

// Sort orders a store's List by the chosen timestamp.
const (
	SortCreated = "created"
	SortUpdated = "updated"
)

// Index declares a secondary index on a top-level document field. The
// scan-based engine enforces Unique on write; others just document fields.
type Index struct {
	// Path is the field name (e.g. "name"). A leading "$." is accepted for parity
	// with the old json_extract paths.
	Path string
	// Unique rejects a write whose value collides with another document.
	Unique bool
	// ActiveOnly means a soft-deleted document does not hold its unique value.
	ActiveOnly bool
}

// Spec configures a store: bucket, not-found sentinel, sort field for List
// ([SortCreated] default, or [SortUpdated]), and indexes.
type Spec struct {
	Bucket   string
	NotFound error
	SortBy   string
	Indexes  []Index
}

// Filter is an equality predicate over a document field, used by ListFiltered.
type Filter struct {
	Path string
	Val  any
}
