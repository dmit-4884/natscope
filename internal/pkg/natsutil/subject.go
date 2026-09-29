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

// ValidateSubjectPattern validates a subscription or mapping pattern: non-empty tokens, whole-token
// wildcards, ">" only as the last token, and no whitespace or control characters.
func ValidateSubjectPattern(subject string) error {
	return validateSubjectTokens(subject, true)
}

// ValidateLiteralSubject validates a publish subject, which must contain no wildcards.
func ValidateLiteralSubject(subject string) error {
	return validateSubjectTokens(subject, false)
}

// validateSubjectTokens runs the token checks shared by patterns and literal subjects.
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

// checkSubjectCharacters rejects whitespace and control characters, which nats.go lets through.
func checkSubjectCharacters(subject string) error {
	for _, r := range subject {
		if r < 0x21 || r == 0x7f {
			return subjectValidationErr(fmt.Sprintf("subject %q contains whitespace or control characters", subject))
		}
	}
	return nil
}

// subjectValidationErr wraps a subject-syntax failure in the domain validation error type.
func subjectValidationErr(description string) error {
	return &errs.NATSValidationError{Description: description}
}

// MatchSubject checks if a subject matches a NATS pattern.
// Wildcards: "*" matches a single token, ">" matches one or more tokens.
func MatchSubject(pattern, subject string) bool {
	return matchSubjectTokens(pattern, strings.Split(subject, "."))
}

// matchSubjectTokens is MatchSubject with the subject already split into tokens.
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
