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
