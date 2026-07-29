// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrProtoMessageNotFound is returned when a proto message type cannot be
	// found.
	ErrProtoMessageNotFound = errors.New("proto: message type not found")
	// ErrNoProtoSources is returned when no proto sources are configured /
	// enabled.
	ErrNoProtoSources = errors.New("proto: no proto sources configured")
	// ErrSchemaConflict means descriptor merge produced an error-level conflict
	// (strict mode rejects compilation).
	ErrSchemaConflict = errors.New("proto: schema conflict")
)
