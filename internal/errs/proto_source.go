// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrProtoSourceNotFound is returned when a proto source cannot be found.
	ErrProtoSourceNotFound = errors.New("proto source: not found")

	// ErrProtoSourceNameAlreadyInUse is returned when source name is already taken.
	ErrProtoSourceNameAlreadyInUse = errors.New("proto source: name already in use")

	// ErrProtoVersionNotFound is returned when a proto version cannot be found.
	ErrProtoVersionNotFound = errors.New("proto version: not found")

	// ErrProtoVersionAlreadyExists is returned when version already exists.
	ErrProtoVersionAlreadyExists = errors.New("proto version: already exists")

	// ErrProtoSelectionNotFound is returned when a proto selection cannot be found.
	ErrProtoSelectionNotFound = errors.New("proto selection: not found")

	// ErrProtoDescriptorNotFound is returned when a proto descriptor cannot be found.
	ErrProtoDescriptorNotFound = errors.New("proto descriptor: not found")
)
