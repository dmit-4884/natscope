// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package publish coordinates the publish flow: optionally proto-encode the
// payload, publish via NATS, record history.
//
// Soft-failure semantics: encode/input/publish errors surface via
// PublishResult.Error (never transport errors).
//
// Request reuses the payload encoding for a core NATS request-reply exchange;
// its failures are plain errors and it keeps no history.
//
// Implementations must be safe for concurrent use.
package publish
