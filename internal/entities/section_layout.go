// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// SectionLayout arranges one sidebar list: Pinned names come first, then
// Order; names in neither follow by name.
type SectionLayout struct {
	// Pinned names, in display order.
	Pinned []string

	// Order holds manually ordered names that are not pinned, in display order.
	Order []string
}
