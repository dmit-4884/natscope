// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package protoutils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
)

// NormalizeRoots canonicalizes root prefixes: backslashes -> slashes, trim
// surrounding slashes/whitespace, drop empties and dupes. First occurrence wins.
func NormalizeRoots(in []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range in {
		r = strings.TrimSpace(strings.ReplaceAll(r, "\\", "/"))
		r = strings.Trim(r, "/")
		if r == "" || r == "." || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	return out
}

// RootsOrigin names the ladder tier that produced the roots.
type RootsOrigin string

const (
	RootsOriginManual   RootsOrigin = "manual"
	RootsOriginBuf      RootsOrigin = "buf"
	RootsOriginInferred RootsOrigin = "inferred"
)

// ResolvedLayout is the deterministic compilation layout for one source.
type ResolvedLayout struct {
	Origin  RootsOrigin
	Roots   []string          // roots; implicit "" never listed
	Srcs    map[string]string // canonical import path -> content
	Targets []string          // sorted compile targets (canonical keys)
	Locks   [][]byte          // raw buf.lock contents, when captured
	Diags   []entities.CompileDiagnostic
}

// ResolveLayout runs the ladder: manual roots, then buf config, then
// import-suffix inference. See docs/auto-walk-audit.md §5.2.
func ResolveLayout(
	files []entities.ProtoFileEntry,
	manualRoots []string,
	configs []entities.ProtoFileEntry,
) *ResolvedLayout {
	if len(files) == 0 {
		return &ResolvedLayout{Diags: []entities.CompileDiagnostic{{
			Severity: entities.DiagnosticError,
			Message:  "no .proto files to compile",
			Hint:     "check the source path and ExcludePrefixes",
		}}}
	}

	locks := collectLocks(configs)

	if roots := NormalizeRoots(manualRoots); len(roots) > 0 {
		srcs, targets, diags := applyRoots(files, roots, true)
		return &ResolvedLayout{
			Origin:  RootsOriginManual,
			Roots:   roots,
			Srcs:    srcs,
			Targets: targets,
			Locks:   locks,
			Diags:   diags,
		}
	}

	if buf, ok := DetectBufLayout(configs); ok {
		srcs, targets, diags := applyRoots(files, buf.Roots, false)
		// Zero usable roots means "module at source root": re-run with
		// fallback=true so root-"" semantics keep files instead of skipping all.
		if len(buf.Roots) == 0 {
			srcs, targets, diags = applyRoots(files, nil, true)
		}
		return &ResolvedLayout{
			Origin:  RootsOriginBuf,
			Roots:   buf.Roots,
			Srcs:    srcs,
			Targets: targets,
			Locks:   buf.Locks,
			Diags:   diags,
		}
	}

	roots, infDiags := inferRoots(files)
	srcs, targets, applyDiags := applyRoots(files, roots, true)
	return &ResolvedLayout{
		Origin: RootsOriginInferred, Roots: roots,
		Srcs: srcs, Targets: targets, Locks: locks,
		Diags: append(infDiags, applyDiags...),
	}
}

func collectLocks(configs []entities.ProtoFileEntry) [][]byte {
	return slices.ToWithFilter(configs,
		func(c entities.ProtoFileEntry) bool { return strings.HasSuffix(c.Path, "buf.lock") },
		func(c entities.ProtoFileEntry) []byte { return []byte(c.Content) },
	)
}

// inferRoots infers import roots from suffix matches: an import satisfied by
// "src/a/b.proto" proves "src" is a root. Ties prefer the deepest path.
func inferRoots(files []entities.ProtoFileEntry) ([]string, []entities.CompileDiagnostic) {
	contentByPath := make(map[string]string, len(files))
	for _, f := range files {
		contentByPath[f.Path] = f.Content
	}

	importSet := map[string]bool{}
	for _, f := range files {
		for _, imp := range ExtractImports(f.Content) {
			importSet[imp] = true
		}
	}
	imports := make([]string, 0, len(importSet))
	for imp := range importSet {
		imports = append(imports, imp)
	}
	sort.Strings(imports) // deterministic order

	rootSet := map[string]bool{}
	var diags []entities.CompileDiagnostic

	for _, imp := range imports {
		if _, exact := contentByPath[imp]; exact {
			continue // import is already source-root-relative; root "" is implicit
		}
		suffix := "/" + imp
		var candidates []string
		for p := range contentByPath {
			if strings.HasSuffix(p, suffix) {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			continue // WKT / BSR dep / genuinely missing — the compiler will report it precisely
		}
		sort.Strings(candidates) // deterministic before choosing
		chosen := chooseDeepest(candidates)
		if len(candidates) > 1 && !allSameContent(candidates, contentByPath) {
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticWarning,
				Message: fmt.Sprintf("import %q matches multiple files (%s); using %q",
					imp, strings.Join(candidates, ", "), chosen),
				Hint: "if the wrong file was chosen, set ImportRoots on the source",
			})
		}
		rootSet[chosen[:len(chosen)-len(suffix)]] = true
	}

	roots := make([]string, 0, len(rootSet))
	for r := range rootSet {
		roots = append(roots, r)
	}
	sort.Strings(roots)
	return roots, diags
}

// chooseDeepest prefers the path with the most segments; ties break to the
// lexicographically smallest (pre-sorted, so first wins).
func chooseDeepest(cands []string) string {
	best := cands[0]
	for _, c := range cands[1:] {
		if strings.Count(c, "/") > strings.Count(best, "/") {
			best = c
		}
	}
	return best
}

func allSameContent(paths []string, contentByPath map[string]string) bool {
	first := contentByPath[paths[0]]
	for _, p := range paths[1:] {
		if contentByPath[p] != first {
			return false
		}
	}
	return true
}

// applyRoots resolves each file's canonical import path by stripping its
// longest matching root; keepOutside=false skips unmatched files (INFO diag).
func applyRoots(
	files []entities.ProtoFileEntry,
	roots []string,
	keepOutside bool,
) (map[string]string, []string, []entities.CompileDiagnostic) {
	sorted := append([]string(nil), roots...)
	sort.Slice(sorted, func(i, j int) bool { return len(sorted[i]) > len(sorted[j]) })

	srcs := make(map[string]string, len(files))
	keyOwner := make(map[string]string, len(files))
	var targets []string
	var diags []entities.CompileDiagnostic

	for _, f := range files {
		key, matched := "", false
		for _, r := range sorted {
			if strings.HasPrefix(f.Path, r+"/") {
				key, matched = f.Path[len(r)+1:], true
				break
			}
		}
		if !matched {
			if !keepOutside {
				diags = append(diags, entities.CompileDiagnostic{
					Severity: entities.DiagnosticInfo,
					File:     f.Path,
					Message:  "file is outside the configured buf module roots; skipped",
				})
				continue
			}
			key = f.Path
		}
		if owner, dup := keyOwner[key]; dup {
			if srcs[key] == f.Content {
				diags = append(diags, entities.CompileDiagnostic{
					Severity: entities.DiagnosticInfo,
					File:     f.Path,
					Message:  fmt.Sprintf("identical duplicate of %s at import path %q; deduplicated", owner, key),
				})
				continue
			}
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				File:     f.Path,
				Message: fmt.Sprintf(
					"import path collision: %q resolves to both %s and %s with different contents",
					key, owner, f.Path,
				),
				Hint: "rename one of the files, or configure ImportRoots/ExcludePrefixes on the source",
			})
			continue
		}
		keyOwner[key] = f.Path
		srcs[key] = f.Content
		targets = append(targets, key)
	}
	sort.Strings(targets)
	return srcs, targets, diags
}
