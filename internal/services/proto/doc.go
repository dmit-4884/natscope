// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package proto is the unified business-logic surface for protobuf operations.
//
// Git sources need a selection to decode; local sources are implicit under
// LocalTag. Reload must stay idempotent — broadcast-driven Reset is live
// consumers' only correctness guarantee on descriptor change.
package proto
