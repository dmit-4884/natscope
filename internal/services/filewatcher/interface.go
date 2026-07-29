// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package filewatcher

import "context"

// ChangeCallback fires (debounced) on possible .proto changes; it re-reads the
// directory itself, since passing contents would race slow compiles against
// fresh snapshots. ctx is the watcher-scoped context, canceled on Stop.
type ChangeCallback func(ctx context.Context, sourceID string)

// Service watches local directories for .proto file changes.
type Service interface {
	// Watch starts watching a directory for .proto file changes.
	Watch(sourceID string, dirPath string) error

	// Unwatch stops watching a directory.
	Unwatch(sourceID string)

	// SetCallback sets the function to call when changes are detected.
	SetCallback(cb ChangeCallback)

	// Start begins the event loop.
	Start() error

	// Stop shuts down the watcher and all watches.
	Stop()
}
