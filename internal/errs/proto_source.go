// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrProtoSourceNotFound is returned when a proto source cannot be found.
	ErrProtoSourceNotFound = errors.New("proto source: not found")

	// ErrProtoSourceNameAlreadyInUse is returned when source name is already taken.
	ErrProtoSourceNameAlreadyInUse = errors.New("proto source: name already in use")

	// ErrProtoRefNotFound is returned when a tag, branch, label or commit does not exist.
	ErrProtoRefNotFound = errors.New("proto ref: not found")

	// ErrProtoFileSetNotFound is returned when no raw files are stored for a revision.
	ErrProtoFileSetNotFound = errors.New("proto file set: not found")

	// ErrProtoDescriptorNotFound is returned when a proto descriptor cannot be found.
	ErrProtoDescriptorNotFound = errors.New("proto descriptor: not found")
)

// RegistryError is a failure a Buf Schema Registry reported; Code is the Connect code, Message is safe to show.
type RegistryError struct {
	Code    string
	Message string
}

// Error implements the error interface.
func (e *RegistryError) Error() string {
	return "buf registry: " + e.Message
}
