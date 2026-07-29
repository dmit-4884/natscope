// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/bufbuild/protocompile/reporter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// collectingReporter captures every protocompile error/warning instead of
// aborting; Error() returns nil so the compiler keeps walking for full diagnostics.
type collectingReporter struct {
	errs     []reporter.ErrorWithPos
	warnings []reporter.ErrorWithPos
}

func (r *collectingReporter) Error(err reporter.ErrorWithPos) error {
	r.errs = append(r.errs, err)
	return nil
}

func (r *collectingReporter) Warning(err reporter.ErrorWithPos) {
	r.warnings = append(r.warnings, err)
}

// summarizeDiagnostics produces a one-line health-badge message: first
// error-severity diagnostic, else a count summary.
func summarizeDiagnostics(diags []entities.CompileDiagnostic) string {
	for _, d := range diags {
		if d.Severity == entities.DiagnosticError && d.Message != "" {
			if d.File != "" {
				return fmt.Sprintf("%s: %s", d.File, d.Message)
			}
			return d.Message
		}
	}
	if len(diags) > 0 {
		return fmt.Sprintf("%d diagnostic(s)", len(diags))
	}
	return ""
}

// errorWithPosToDiagnostic converts a protocompile diagnostic into a
// CompileDiagnostic, surfacing "file not found" as MissingImport with a hint.
func errorWithPosToDiagnostic(
	e reporter.ErrorWithPos,
	sev entities.DiagnosticSeverity,
	includeDirs []string,
) entities.CompileDiagnostic {
	pos := e.GetPosition()
	underlying := e.Unwrap()
	msg := underlying.Error()

	d := entities.CompileDiagnostic{
		Severity: sev,
		File:     pos.Filename,
		Line:     pos.Line,
		Column:   pos.Col,
		Message:  msg,
	}

	if imp := extractMissingImport(msg); imp != "" {
		d.MissingImport = imp
		switch {
		case strings.HasPrefix(imp, "google/protobuf/"):
			// Files type is strict: well-known types are not auto-included, user must
			// list them explicitly.
			d.Hint = fmt.Sprintf(
				"Files type is strict: well-known types are not auto-included. "+
					"Add %q to Files, or add the directory containing it to Include Directories.",
				imp,
			)
		case len(includeDirs) == 0:
			d.Hint = fmt.Sprintf(
				"add a directory containing %q to Include Directories, or add the file path to Files",
				imp,
			)
		default:
			d.Hint = fmt.Sprintf(
				"%q was not found under any of the configured Include Directories — add another directory that contains it, or add the file path to Files",
				imp,
			)
		}
	}

	return d
}

// extractMissingImport pulls the import path from an unresolved-import
// diagnostic ("" if unmatched); handles protocompile's two v0.14.x error formats.
func extractMissingImport(msg string) string {
	if rest, ok := strings.CutPrefix(msg, "file not found: "); ok {
		return strings.TrimSpace(rest)
	}
	const prefix = `could not resolve path "`
	if i := strings.Index(msg, prefix); i >= 0 {
		rest := msg[i+len(prefix):]
		if j := strings.Index(rest, `"`); j > 0 {
			return rest[:j]
		}
	}
	return ""
}

// autoWalkHint builds a missing-import hint for the auto-walk flow, covering
// mixed import styles, shadowed well-known types, or a genuinely absent file.
func autoWalkHint(missing string, layout *protoutils.ResolvedLayout) string {
	base := path.Base(missing)
	var sameBase []string
	for key := range layout.Srcs {
		if path.Base(key) == base {
			sameBase = append(sameBase, key)
		}
	}
	sort.Strings(sameBase)

	switch {
	case len(sameBase) > 0:
		roots := strings.Join(layout.Roots, ", ")
		if roots == "" {
			roots = "<source root>"
		}
		return fmt.Sprintf(
			"file exists at import path %q (detected roots: %s, origin: %s) — "+
				"the repo mixes import styles; rewrite the import relative to a root, or set ImportRoots on the source",
			sameBase[0], roots, layout.Origin)
	case strings.HasPrefix(missing, "google/protobuf/"):
		return "well-known types resolve automatically; if this fails, a vendored copy may be shadowing them"
	default:
		return fmt.Sprintf(
			"%q was not found in the source (origin: %s) — vendor it, add the buf dependency to buf.lock, or set ImportRoots",
			missing, layout.Origin)
	}
}
