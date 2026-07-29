// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package errs

import "errors"

// ErrMessageTemplateNotFound means no template has the requested id (or it was
// already deleted).
var ErrMessageTemplateNotFound = errors.New("message template: not found")
