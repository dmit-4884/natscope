// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// CompileOutcome is the result of one compile pass.
type CompileOutcome struct {
	Valid           bool
	MessageTypes    int
	FileDescriptors int
	Diagnostics     []CompileDiagnostic
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
