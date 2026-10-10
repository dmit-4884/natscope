// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// SidebarLayoutUpdate replaces the sections that are set and keeps the others.
type SidebarLayoutUpdate struct {
	// ConnectionID is the saved connection whose layout changes.
	ConnectionID string `normalize:"trim"`

	Streams *SectionLayout
	KV      *SectionLayout
	Objects *SectionLayout
}
