// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// SidebarLayout is how the sidebar arranges one saved connection's streams,
// KV buckets and object buckets.
type SidebarLayout struct {
	// ConnectionID is the saved connection the layout belongs to.
	ConnectionID string

	Streams SectionLayout
	KV      SectionLayout
	Objects SectionLayout

	// UpdatedAt is when the layout last changed; zero until it is first saved.
	UpdatedAt time.Time
}
