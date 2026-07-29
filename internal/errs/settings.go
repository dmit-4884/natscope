// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

var (
	// ErrSettingsNotFound is returned when user settings cannot be found.
	ErrSettingsNotFound = errors.New("settings: not found")
)
