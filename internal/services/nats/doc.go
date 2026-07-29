// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package nats is the business-logic surface for NATS / JetStream operations;
// connection IDs are SavedConnection.Id, pooled and connected lazily
// (singleflight on the cold path). Implementations must be safe for concurrent
// use.
package nats
