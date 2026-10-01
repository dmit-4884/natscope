// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import "time"

// File-local named constants for proto-service magic numbers.
const (
	// estimatedMessagesPerSnapshot is the initial slice capacity hint when
	// flattening descriptor sets.
	estimatedMessagesPerSnapshot = 64

	// minActiveSnapshotsForConflictReport is the smallest active set before
	// MergeWithReport has anything cross-source to compare.
	minActiveSnapshotsForConflictReport = 2

	// fileWatcherDebounceTimeout caps a local source's descriptor rebuild after a
	// filesystem nudge; safety net for pathological disks.
	fileWatcherDebounceTimeout = 60 * time.Second

	// LocalRevision is the revision of a local directory source's schema.
	LocalRevision = "local"

	// FilesRevision is the revision of a Files-type source's schema.
	FilesRevision = "files"

	startupCompileTimeout = 5 * time.Minute
)
