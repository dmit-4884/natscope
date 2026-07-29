// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import "strings"

// IsInternalSubject reports whether a subject belongs to NATS itself: the "$"
// system namespaces ($JS, $SYS) and the "_" inbox namespace. Application
// subjects are never filtered here.
func IsInternalSubject(subject string) bool {
	if len(subject) == 0 {
		return true
	}
	return subject[0] == '$' || subject[0] == '_'
}

// MatchSubject checks if a subject matches a NATS pattern.
// Wildcards: "*" matches a single token, ">" matches one or more tokens.
func MatchSubject(pattern, subject string) bool {
	patternParts := strings.Split(pattern, ".")
	subjectParts := strings.Split(subject, ".")

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
