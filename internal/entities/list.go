// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"time"

	"github.com/altessa-s/go-atlas/core/types/ptr"
)

// DefaultListLimit is the default number of items per page.
const DefaultListLimit int64 = 50

// MaxListLimit caps a client-supplied page size so a single request can't scan
// and return an unbounded slice.
const MaxListLimit int64 = 500

// ListItems is a constraint for List item types.
type ListItems any

// ListBase provides common pagination params; embed in list request entities.
type ListBase struct {
	// Cursor for cursor-based pagination.
	Cursor string

	// Limit caps returned items.
	Limit *int64

	// IncludeTotalCount includes total count in response.
	IncludeTotalCount bool
}

// GetLimit returns the page size: the default when unset or non-positive,
// clamped to MaxListLimit.
func (l *ListBase) GetLimit() int64 {
	limit := ptr.Unwrap(l.Limit, DefaultListLimit)
	switch {
	case limit <= 0:
		return DefaultListLimit
	case limit > MaxListLimit:
		return MaxListLimit
	default:
		return limit
	}
}

// List is a paginated list response.
type List[T ListItems] struct {
	// Items is the result slice.
	Items T

	// Total is the count before pagination; -1 or nil means not computed.
	Total *int64

	// NextCursor points to the next page. Nil if this is the last page.
	NextCursor *string
}

// SoftDelete carries data for soft-deleting an entity.
type SoftDelete struct {
	Id           string
	NewUpdatedAt time.Time
}
