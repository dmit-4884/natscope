// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package history records every publish attempt (success or failure) for the
// UI's history panel: Record stores an outcome, List paginates.
//
// History writes are non-fatal to the publish path — log and move on.
//
// Implementations must be safe for concurrent use.
package history
