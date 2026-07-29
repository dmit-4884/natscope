// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrInvalidRequest is returned when the request is invalid or malformed.
	ErrInvalidRequest = errors.New("invalid request")

	// ErrNotFound is returned when a resource cannot be found.
	ErrNotFound = errors.New("not found")

	// ErrDuplicateEntity is returned when an entity already exists.
	ErrDuplicateEntity = errors.New("entity already exists")

	// ErrInvalidListCursor is returned when pagination cursor is invalid.
	ErrInvalidListCursor = errors.New("invalid list cursor")

	// ErrPermissionDenied is returned when user lacks permission.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrUnauthorized is returned when user is not authenticated.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrUnexpectedSingleflightType is returned when a singleflight group
	// yields a value of an unexpected dynamic type (internal invariant).
	ErrUnexpectedSingleflightType = errors.New("singleflight returned unexpected type")
)
