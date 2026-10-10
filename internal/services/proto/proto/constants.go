// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package proto

import "time"

// File-local named constants for proto-service magic numbers.
const (
	// minActiveSnapshotsForConflictReport is the smallest active set before
	// FindConflicts has anything cross-source to compare.
	minActiveSnapshotsForConflictReport = 2

	// maxStoredRevisions bounds the schemas kept per source, besides the active and pinned ones.
	maxStoredRevisions = 20

	// fileWatcherDebounceTimeout caps a local source's descriptor rebuild after a
	// filesystem nudge; safety net for pathological disks.
	fileWatcherDebounceTimeout = 60 * time.Second

	// LocalRevision is the revision of a local directory source's schema.
	LocalRevision = "local"

	startupCompileTimeout = 5 * time.Minute
)
