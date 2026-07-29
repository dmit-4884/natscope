// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

// containsWildcard checks if a subject pattern contains NATS wildcards.
func containsWildcard(pattern string) bool {
	for _, c := range pattern {
		if c == '*' || c == '>' {
			return true
		}
	}
	return false
}
