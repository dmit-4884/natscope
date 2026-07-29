// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
)

// File-size hard caps applied by ReadFilesFromPaths.
const (
	maxFilesProtoSize = 5 * 1024 * 1024  // 5 MiB per .proto file
	maxFilesTotalSize = 20 * 1024 * 1024 // 20 MiB combined
)

// ReadFilesFromPaths reads each path and returns ProtoFileEntry items keyed
// by basename (import-resolution key); failures become diagnostics, not errors.
func ReadFilesFromPaths(paths []string) ([]entities.ProtoFileEntry, []entities.CompileDiagnostic) {
	entries := make([]entities.ProtoFileEntry, 0, len(paths))
	var (
		diags []entities.CompileDiagnostic
		total int64
	)

	seen := make(map[string]string, len(paths)) // basename -> source path

	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  "path must be absolute",
				Hint:     "use a fully qualified path like /Users/you/proto/myapi.proto",
			})
			continue
		}
		if !strings.HasSuffix(strings.ToLower(p), ".proto") {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  "not a .proto file",
			})
			continue
		}

		info, err := os.Stat(p)
		if err != nil {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  fmt.Sprintf("file not accessible: %v", err),
			})
			continue
		}
		if info.IsDir() {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  "path is a directory; only .proto files are accepted in Files",
				Hint:     "use Include Directories for directories of dependencies",
			})
			continue
		}
		if info.Size() > maxFilesProtoSize {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  fmt.Sprintf("file too large: %d bytes (max %d)", info.Size(), maxFilesProtoSize),
			})
			continue
		}
		total += info.Size()
		if total > maxFilesTotalSize {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  fmt.Sprintf("combined input exceeds %d bytes", maxFilesTotalSize),
			})
			break
		}

		bn := filepath.Base(p)
		if prev, dup := seen[bn]; dup {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  fmt.Sprintf("duplicate file basename %q (already added from %s)", bn, prev),
				Hint:     "rename one of the files or place its directory under Include Directories instead",
			})
			continue
		}
		seen[bn] = p

		content, err := os.ReadFile(p)
		if err != nil {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     p,
				Message:  fmt.Sprintf("read failed: %v", err),
			})
			continue
		}

		entries = append(entries, entities.ProtoFileEntry{
			Path:    bn,
			Content: string(content),
			Size:    info.Size(),
		})
	}

	return entries, diags
}

// ValidateIncludeDirs verifies each include dir exists and is a directory,
// reporting failures as diagnostics.
func ValidateIncludeDirs(dirs []string) []entities.CompileDiagnostic {
	var diags []entities.CompileDiagnostic
	for _, d := range dirs {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if !filepath.IsAbs(d) {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     d,
				Message:  "include directory path must be absolute",
			})
			continue
		}
		info, err := os.Stat(d)
		if err != nil {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     d,
				Message:  fmt.Sprintf("include directory not accessible: %v", err),
			})
			continue
		}
		if !info.IsDir() {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     d,
				Message:  "include path is not a directory",
			})
			continue
		}
	}
	return diags
}
