// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package live is the live-subscription service surface: one session per client
// viewing real-time messages (subscribe, batch, rate-limit, decode, emit stats,
// tear down).
//
// BroadcastProtoReload flips a dirty bit on every session so each re-inits its
// proto decoder before the next message; wired in fx/transport.go to proto's
// reload callback.
//
// Implementations must be safe for concurrent use across goroutines.
package live
