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

	// LocalTag is the synthetic tag for descriptors from a local-filesystem source
	// (no git versions).
	LocalTag = "local"
)
