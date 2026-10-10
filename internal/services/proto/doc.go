// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

// Package proto is the unified business-logic surface for protobuf operations.
//
// Git and BSR sources decode with their selected ref, uploads with their latest version and local
// sources with LocalRevision. Reload must stay idempotent: broadcast-driven Reset is live consumers'
// only correctness guarantee on a schema change.
package proto
