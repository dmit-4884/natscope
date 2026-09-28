// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"fmt"

	"github.com/dmit-4884/natscope/internal/errs"
)

// isHeaderNameByte reports whether b is a valid RFC 7230 "token" character,
// the character class NATS headers use (they are serialized as HTTP-style
// headers over the wire).
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

// ValidateHeaderName reports whether name is a valid RFC 7230 token. NATS
// serializes headers as HTTP-style headers; a name outside this set is
// silently dropped on encode (QA-079/QA-162) instead of reaching the peer.
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

// ValidateHeaderNames rejects the first header whose key is not a valid
// RFC 7230 token, so publishers get an immediate error instead of a silently
// incomplete header set on the wire.
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
