// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package messages coordinates list/get of JetStream messages across NATS
// (fetch), proto (decode) and settings (defaults).
//
// Settings are advisory: missing/unreadable ones fall through to NATS defaults.
// Decode failures never abort the response — messages return with DecodeError
// populated.
//
// Implementations must be safe for concurrent use across goroutines.
package messages
