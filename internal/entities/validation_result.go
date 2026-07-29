// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ValidationViolation is one protovalidate constraint failure.
type ValidationViolation struct {
	FieldPath    string
	Message      string
	ConstraintId string
}

// ValidationResult is the protovalidate outcome for one message.
type ValidationResult struct {
	Valid      bool
	Violations []*ValidationViolation
	Error      string
}
