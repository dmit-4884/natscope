// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package errs

import "errors"

// ErrMessageTemplateNotFound means no template has the requested id (or it was
// already deleted).
var ErrMessageTemplateNotFound = errors.New("message template: not found")

// ErrMessageTemplateNameRequired is returned when a template name is empty or whitespace-only.
var ErrMessageTemplateNameRequired = errors.New("message template: name is required")
