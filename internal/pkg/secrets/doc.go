// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package secrets stores per-entity secret values outside the database, in the
// OS keychain. The database keeps only non-secret fields; secrets (NATS auth,
// TLS private keys, git tokens) live here, addressed by namespace and entity id.
// [Memory] is an in-memory implementation for tests.
package secrets
