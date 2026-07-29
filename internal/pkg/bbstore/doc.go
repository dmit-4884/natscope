// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package bbstore is a JSON-document store over a bbolt key-value database. It
// is the bbolt counterpart of the former SQLite docstore: each store is one
// top-level bucket keyed by document id, with the document held as a JSON value.
//
// A storage maps its domain entity to and from a persistence model that embeds
// [Base], and declares the fields it queries by through [Spec]. The store offers
// the CRUD, keyset List, and equality lookups every domain needs. Secrets do not
// belong here; they live in the OS keychain (internal/pkg/secrets).
//
// The engine is scan-based: secondary lookups and uniqueness checks walk the
// bucket. That is fine for the data sizes natscope holds locally. The bucket key
// is the id; the value is the JSON document, whose tags match the old SQLite
// format so existing files migrate without reshaping.
package bbstore
