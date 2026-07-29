// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package mappings is the subject-mapping service surface: binds NATS subject
// patterns (`*`/`>` wildcards) to proto message types within a source, with
// CRUD, BulkSave, GetAll, and a Resolver.
//
// Resolver is an immutable matcher snapshot for the concurrent hot path; the
// pointer is swapped atomically on every mutation, so holders of an old
// resolver stay consistent until they re-read. Safe for concurrent use.
package mappings
