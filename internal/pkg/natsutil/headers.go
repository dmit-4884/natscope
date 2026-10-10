// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsutil

import (
	"fmt"

	"github.com/dmit-4884/natscope/internal/errs"
)

// isHeaderNameByte reports whether b is an RFC 7230 token character.
func isHeaderNameByte(b byte) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '!' || b == '#' || b == '$' || b == '%' || b == '&' || b == '\'' ||
		b == '*' || b == '+' || b == '-' || b == '.' || b == '^' || b == '_' ||
		b == '`' || b == '|' || b == '~':
		return true
	default:
		return false
	}
}

// ValidateHeaderName reports whether name is a valid RFC 7230 token.
func ValidateHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !isHeaderNameByte(name[i]) {
			return false
		}
	}
	return true
}

// ValidateHeaderNames returns an error for the first header key that is not a valid RFC 7230 token.
func ValidateHeaderNames(headers map[string]string) error {
	for name := range headers {
		if !ValidateHeaderName(name) {
			return &errs.NATSValidationError{
				Description: fmt.Sprintf("header name %q is not a valid header token (RFC 7230)", name),
			}
		}
	}
	return nil
}
