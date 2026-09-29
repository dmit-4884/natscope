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

// SubjectsCollide reports whether some literal subject matches both patterns, like nats-server's
// SubjectsCollide: "*" is compatible with any one token and ">" with one or more.
func SubjectsCollide(a, b string) bool {
	if a == b {
		return true
	}
	aTokens, bTokens := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(aTokens) && i < len(bTokens); i++ {
		at, bt := aTokens[i], bTokens[i]
		if at == ">" || bt == ">" {
			return true
		}
		if at != bt && at != "*" && bt != "*" {
			return false
		}
	}
	return len(aTokens) == len(bTokens)
}

// TransformDestinationPattern turns a subject-transform destination into a pattern matching every
// subject it can produce: "$1", {{wildcard(1)}}, {{partition(...)}} and {{random(...)}} yield one
// token ("*"), while split and slice functions yield any number of them, so the rest becomes ">".
func TransformDestinationPattern(dest string) string {
	tokens := strings.Split(dest, ".")
	for i, token := range tokens {
		if !isTransformFunction(token) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(token, "{{")))
		if strings.HasPrefix(name, "split") || strings.HasPrefix(name, "slice") {
			return strings.Join(append(tokens[:i:i], ">"), ".")
		}
		tokens[i] = "*"
	}
	return strings.Join(tokens, ".")
}

func isTransformFunction(token string) bool {
	if strings.Contains(token, "{{") {
		return true
	}
	if len(token) < 2 || token[0] != '$' {
		return false
	}
	for _, r := range token[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
