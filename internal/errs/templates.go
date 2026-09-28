// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

// ErrMessageTemplateNotFound means no template has the requested id (or it was
// already deleted).
var ErrMessageTemplateNotFound = errors.New("message template: not found")

// ErrMessageTemplateNameRequired is returned when a template name is empty or
// whitespace-only. buf.validate checks min_len on the raw request, so a
// whitespace-only name must be rejected again after normalization trims it.
var ErrMessageTemplateNameRequired = errors.New("message template: name is required")
