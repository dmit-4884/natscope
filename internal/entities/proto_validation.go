// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// RepositoryValidation is the outcome of probing a Git repo for accessibility;
// Error set on failure.
type RepositoryValidation struct {
	Valid bool
	Error *string
}

// LocalPathValidation is the outcome of probing a local proto-source dir; Valid
// requires a dir with ≥1 .proto file.
type LocalPathValidation struct {
	Valid          bool
	ProtoFileCount int
	Error          *string
}
