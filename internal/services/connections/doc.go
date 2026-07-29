// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package connections is the saved-NATS-connections service surface: CRUD,
// duplication, and saved-config probing with probe-outcome persisted to Meta.
//
// Mutations must evict the live pool connection (next use re-dials), and
// telemetry-only writes (probe-result Meta) must not bump timestamps/ETag.
// Implementations must be safe for concurrent use.
package connections
