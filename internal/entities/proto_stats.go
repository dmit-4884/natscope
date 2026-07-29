// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoStats holds the count of loaded message types and any load error.
type ProtoStats struct {
	MessagesCount int
	Error         string
}

// IsLoaded reports whether any proto messages are loaded; nil-safe.
func (p *ProtoStats) IsLoaded() bool {
	return p != nil && p.MessagesCount > 0
}

// ProtoLoadResult is the outcome of fetch+compile across all active proto
// selections.
type ProtoLoadResult struct {
	// MessageCount is the total message types available across all active
	// snapshots.
	MessageCount int
	// CompiledCount is the number of selections that compiled successfully.
	CompiledCount int
	// FailedCount is the number of selections that failed to compile (per-source
	// errors logged at WARN).
	FailedCount int
}
