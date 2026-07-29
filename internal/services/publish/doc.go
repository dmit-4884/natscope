// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package publish coordinates the publish flow: optionally proto-encode the
// payload, publish via NATS, record history.
//
// Soft-failure semantics: encode/input/publish errors surface via
// PublishResult.Error (never transport errors); history is written for both
// success and failure.
//
// Implementations must be safe for concurrent use.
package publish
