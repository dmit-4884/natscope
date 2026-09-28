// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"fmt"
	"strings"

	"github.com/dmit-4884/natscope/internal/errs"
)

// IsInternalSubject reports whether a subject belongs to NATS itself: the "$"
// system namespaces ($JS, $SYS) and the "_" inbox namespace. Application
// subjects are never filtered here.
func IsInternalSubject(subject string) bool {
	if len(subject) == 0 {
		return true
	}
	return subject[0] == '$' || subject[0] == '_'
}

// ValidateSubjectPattern validates a NATS subject pattern used for
// subscriptions or mappings: tokens must be non-empty, "*"/">" may only
// appear as a whole token, and ">" may only be the last token. Whitespace and
// control characters are rejected. Length limits are enforced elsewhere
// (server subject/control-line caps).
func ValidateSubjectPattern(subject string) error {
	return validateSubjectTokens(subject, true)
}

// ValidateLiteralSubject validates a NATS subject used for publishing: no
// wildcard tokens are allowed, since a literal publish subject must name a
// single subject, never a pattern.
func ValidateLiteralSubject(subject string) error {
	return validateSubjectTokens(subject, false)
}

// validateSubjectTokens implements the shared token-level checks for both
// subscription patterns (wildcards allowed) and literal publish subjects.
func validateSubjectTokens(subject string, allowWildcards bool) error {
	if subject == "" {
		return subjectValidationErr("subject must not be empty")
	}
	if err := checkSubjectCharacters(subject); err != nil {
		return err
	}

	tokens := strings.Split(subject, ".")
	for i, tok := range tokens {
		switch {
		case tok == "":
			return subjectValidationErr(fmt.Sprintf("subject %q has an empty token", subject))
		case tok == ">":
			if !allowWildcards {
				return subjectValidationErr(fmt.Sprintf("subject %q must not contain wildcards", subject))
			}
			if i != len(tokens)-1 {
				return subjectValidationErr(fmt.Sprintf("subject %q: %q must be the last token", subject, ">"))
			}
		case tok == "*":
			if !allowWildcards {
				return subjectValidationErr(fmt.Sprintf("subject %q must not contain wildcards", subject))
			}
		case strings.ContainsAny(tok, "*>"):
			return subjectValidationErr(fmt.Sprintf("subject %q: %q must occupy a whole token", subject, tok))
		}
	}
	return nil
}

// checkSubjectCharacters rejects whitespace and control characters, which
// nats.go accepts client-side but the server either rejects (closing the
// shared pool connection, QA-030) or silently stores as literal bytes.
func checkSubjectCharacters(subject string) error {
	for _, r := range subject {
		if r < 0x21 || r == 0x7f {
			return subjectValidationErr(fmt.Sprintf("subject %q contains whitespace or control characters", subject))
		}
	}
	return nil
}

// subjectValidationErr wraps a subject-syntax failure as the domain
// validation type; transports map it to InvalidArgument without a per-caller
// sentinel table.
func subjectValidationErr(description string) error {
	return &errs.NATSValidationError{Description: description}
}

// MatchSubject checks if a subject matches a NATS pattern.
// Wildcards: "*" matches a single token, ">" matches one or more tokens.
func MatchSubject(pattern, subject string) bool {
	return matchSubjectTokens(pattern, strings.Split(subject, "."))
}

// matchSubjectTokens is MatchSubject with the subject already split, so a
// caller matching one subject against many patterns (MappingResolver.Resolve)
// splits the subject once instead of once per pattern.
func matchSubjectTokens(pattern string, subjectParts []string) bool {
	patternParts := strings.Split(pattern, ".")

	pi, si := 0, 0
	for pi < len(patternParts) && si < len(subjectParts) {
		p := patternParts[pi]
		if p == ">" {
			return true // > matches everything remaining
		}
		if p != "*" && p != subjectParts[si] {
			return false
		}
		pi++
		si++
	}

	return pi == len(patternParts) && si == len(subjectParts)
}
