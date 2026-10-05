// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrSavedConnectionNotFound is returned when a saved connection cannot be found.
	ErrSavedConnectionNotFound = errors.New("connection: not found")

	// ErrConnectionNameAlreadyInUse is returned when connection name is already taken for user.
	ErrConnectionNameAlreadyInUse = errors.New("connection: name already in use")

	// ErrConnectionNameRequired is returned when a connection name is empty or
	// whitespace-only. buf.validate checks min_len on the raw request, so a
	// whitespace-only name must be rejected again after normalization trims it.
	ErrConnectionNameRequired = errors.New("connection: name is required")

	// ErrConnectionURLCredentialsMixed is returned when a connection's server URLs
	// embed different credentials, leaving no single set to lift into auth.
	ErrConnectionURLCredentialsMixed = errors.New("connection: server URLs embed different credentials")

	// ErrConnectionURLCredentialsConflict is returned when server URLs embed
	// credentials while the request also sets an explicit auth config.
	ErrConnectionURLCredentialsConflict = errors.New("connection: credentials set both in the server URL and in auth")

	// ErrConnectionURLInvalid is returned when a server URL contains "@" but cannot be parsed.
	ErrConnectionURLInvalid = errors.New("connection: server URL is invalid")

	// ErrConnectionReadOnly is returned when a write is attempted through a read-only connection.
	ErrConnectionReadOnly = errors.New("connection: read-only")

	// ErrCliContextsHostDisabled is returned when the host's nats CLI contexts are requested while other machines may reach Natscope.
	ErrCliContextsHostDisabled = errors.New("connection: this host's nats CLI contexts are off while other machines may reach Natscope")

	// ErrSidebarLayoutNotFound is returned by [storages/layouts] when nothing was
	// saved for a connection; the connections service reads it as an empty layout.
	ErrSidebarLayoutNotFound = errors.New("connection: sidebar layout not found")
)
