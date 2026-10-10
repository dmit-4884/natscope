// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package errs

import "errors"

var (
	// ErrProtoMessageNotFound is returned when a proto message type cannot be
	// found.
	ErrProtoMessageNotFound = errors.New("proto: message type not found")
	// ErrProtoTypeNotFound is returned when no message, enum or service has the requested name.
	ErrProtoTypeNotFound = errors.New("proto: type not found")
	// ErrNoProtoSources is returned when no proto sources are configured /
	// enabled.
	ErrNoProtoSources = errors.New("proto: no proto sources configured")
	// ErrSchemaConflict means descriptor merge produced an error-level conflict
	// (strict mode rejects compilation).
	ErrSchemaConflict = errors.New("proto: schema conflict")
)

// ProtoEncodeError is a JSON payload that could not be encoded to its protobuf
// message type; Description is user-facing and safe to return to the client.
type ProtoEncodeError struct {
	Description string
}

// Error implements the error interface.
func (e *ProtoEncodeError) Error() string {
	if e == nil || e.Description == "" {
		return "proto: payload could not be encoded"
	}
	return e.Description
}
