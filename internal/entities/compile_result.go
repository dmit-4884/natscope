// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// CompileResult holds the result of a proto compilation operation.
type CompileResult struct {
	MessageTypes    int
	FileDescriptors int
}

// DiagnosticSeverity classifies a compile diagnostic.
type DiagnosticSeverity string

const (
	DiagnosticError   DiagnosticSeverity = "error"
	DiagnosticWarning DiagnosticSeverity = "warning"
	// DiagnosticInfo marks non-actionable notes (dedup/skipped files); never
	// blocks compile.
	DiagnosticInfo DiagnosticSeverity = "info"
)

// CompileDiagnostic is a single error/warning from proto compilation;
// Line/Column are 0 for positionless errors.
type CompileDiagnostic struct {
	Severity DiagnosticSeverity
	File     string
	Line     int
	Column   int
	Message  string
	// MissingImport holds the requested import path when the diagnostic is "import
	// not found".
	MissingImport string
	// Hint is an optional human-readable suggestion.
	Hint string
}
