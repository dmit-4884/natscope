// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/reporter"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// compileFileEntries runs protocompile against the file map plus include dirs.
// Compile errors return nil descriptors; third return is for real failures.
func compileFileEntries(
	ctx context.Context,
	files []entities.ProtoFileEntry,
	includeDirs []string,
) ([]protoreflect.FileDescriptor, []entities.CompileDiagnostic, error) {
	if len(files) == 0 {
		return nil, []entities.CompileDiagnostic{{
			Severity: entities.DiagnosticError,
			Message:  "no .proto files to compile",
			Hint:     "add at least one path to Files",
		}}, nil
	}

	srcs := maps.FromSliceWith(files, func(f entities.ProtoFileEntry) (string, string) {
		return f.Path, f.Content
	})
	targets := slices.To(files, func(f entities.ProtoFileEntry) string { return f.Path })

	// Strict mode: no implicit well-known types — only Files (memResolver) and
	// Include Dirs (diskResolver) resolve; unlisted imports surface as missing.
	memResolver := &protocompile.SourceResolver{
		Accessor: protocompile.SourceAccessorFromMap(srcs),
	}
	diskResolver := newSafeDiskResolver(includeDirs)
	resolver := protocompile.CompositeResolver{
		memResolver,
		diskResolver,
	}

	rep := &collectingReporter{}
	compiler := protocompile.Compiler{
		Resolver:       resolver,
		SourceInfoMode: protocompile.SourceInfoNone,
		Reporter:       rep,
	}

	compiled, compileErr := compiler.Compile(ctx, targets...)

	// Reporter captures parse/link errors but not import-resolve errors (those
	// arrive via compileErr as ErrorWithPos); handle both paths.
	diags := make([]entities.CompileDiagnostic, 0, len(rep.errs)+len(rep.warnings)+1)
	for _, e := range rep.errs {
		diags = append(diags, errorWithPosToDiagnostic(e, entities.DiagnosticError, includeDirs))
	}
	for _, w := range rep.warnings {
		diags = append(diags, errorWithPosToDiagnostic(w, entities.DiagnosticWarning, includeDirs))
	}

	if compileErr != nil && !errors.Is(compileErr, reporter.ErrInvalidSource) {
		// Resolver/import-resolve errors arrive as ErrorWithPos.
		if ewp, ok := errors.AsType[reporter.ErrorWithPos](compileErr); ok {
			diags = append(diags, errorWithPosToDiagnostic(ewp, entities.DiagnosticError, includeDirs))
		} else {
			// Non-positioned error (ctx canceled, panic) — surface via third return so
			// callers distinguish it from proto-level diagnostics.
			if len(diags) == 0 {
				return nil, nil, compileErr
			}
			diags = append(diags, entities.CompileDiagnostic{
				Severity: entities.DiagnosticError,
				Message:  compileErr.Error(),
			})
		}
	}

	// Warnings don't fail compilation; only error-severity diagnostics block
	// returning descriptors.
	if slices.Any(diags, func(d entities.CompileDiagnostic) bool {
		return d.Severity == entities.DiagnosticError
	}) {
		return nil, diags, nil
	}

	// Return warning diags too: UI shows "Compiled OK + N warnings".
	result := slices.To(compiled, func(f linker.File) protoreflect.FileDescriptor { return f })
	return result, diags, nil
}

// ValidateFiles compiles a stored Files-type source or an inline list
// without persisting; supply exactly one of sourceID or files+includeDirs.
func (s *Service) ValidateFiles(
	ctx context.Context,
	sourceID *string,
	files []string,
	includeDirs []string,
) (*entities.CompileOutcome, error) {
	if sourceID != nil && *sourceID != "" {
		src, err := s.sourcesStorage.Get(ctx, *sourceID, false)
		if err != nil {
			return nil, err
		}
		if src.SourceType != entities.SourceTypeFiles {
			return nil, fmt.Errorf("%w: validate files is only available for files-type sources", errs.ErrInvalidRequest)
		}
		files = src.Files
		includeDirs = src.IncludeDirs
	}

	fds, diags, err := readAndCompileFiles(ctx, files, includeDirs)
	if err != nil {
		return nil, err
	}
	if len(fds) == 0 {
		return &entities.CompileOutcome{Diagnostics: diags}, nil
	}
	return &entities.CompileOutcome{
		Valid:           true,
		MessageTypes:    len(s.extractTypes(fds)),
		FileDescriptors: len(fds),
		Diagnostics:     diags,
	}, nil
}

func (s *Service) compileFiles(ctx context.Context, src *entities.ProtoSource) (*entities.CompileOutcome, error) {
	unlock := s.compileLocks.lock(src.Id)
	defer unlock()

	fds, diags, err := readAndCompileFiles(ctx, src.Files, src.IncludeDirs)
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, len(src.Files), diags, nil, "", nil)
		return nil, err
	}
	if len(fds) == 0 {
		s.recordCompile(ctx, src.Id, false, summarizeDiagnostics(diags), 0, len(src.Files), diags, nil, "", nil)
		return &entities.CompileOutcome{Diagnostics: diags}, nil
	}

	d, err := s.storeSchema(ctx, src.Id, FilesRevision, fds)
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, len(fds), diags, nil, "", nil)
		return nil, err
	}
	s.recordCompile(ctx, src.Id, true, "", len(d.MessageTypes), len(fds), diags, nil, "", schemaRevision(d, true))

	s.logger.InfoContext(ctx, "compiled files proto source",
		slog.String("source_id", src.Id),
		slog.Int("files", len(src.Files)),
		slog.Int("include_dirs", len(src.IncludeDirs)),
		slog.Int("message_types", len(d.MessageTypes)),
		slog.Int("file_descriptors", len(fds)))

	s.notifyReload(ctx)

	return &entities.CompileOutcome{
		Valid:           true,
		MessageTypes:    len(d.MessageTypes),
		FileDescriptors: len(fds),
		Diagnostics:     diags,
	}, nil
}

func readAndCompileFiles(
	ctx context.Context,
	files, includeDirs []string,
) ([]protoreflect.FileDescriptor, []entities.CompileDiagnostic, error) {
	entries, diags := protoutils.ReadFilesFromPaths(files)
	diags = append(diags, protoutils.ValidateIncludeDirs(includeDirs)...)
	if len(entries) == 0 || slices.Any(diags, func(d entities.CompileDiagnostic) bool {
		return d.Severity == entities.DiagnosticError
	}) {
		return nil, diags, nil
	}

	fds, compileDiag, err := compileFileEntries(ctx, entries, includeDirs)
	diags = append(diags, compileDiag...)
	return fds, diags, err
}

// safeDiskResolver confines Include Directories imports to relative ".proto" paths under the configured
// directories and never follows a symlink.
type safeDiskResolver struct {
	inner *protocompile.SourceResolver
}

// newSafeDiskResolver builds the Include Directories resolver used by compileFiles.
func newSafeDiskResolver(includeDirs []string) protocompile.Resolver {
	return &safeDiskResolver{inner: &protocompile.SourceResolver{
		ImportPaths: includeDirs,
		Accessor:    safeDiskAccessor,
	}}
}

// FindFileByPath validates the import path before delegating to the wrapped resolver.
func (r *safeDiskResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	if err := validateImportPath(path); err != nil {
		return protocompile.SearchResult{}, err
	}
	return r.inner.FindFileByPath(path)
}

// validateImportPath rejects absolute paths, non-.proto extensions and "../" traversal.
func validateImportPath(p string) error {
	if p == "" {
		return errors.New("empty import path")
	}
	if filepath.IsAbs(p) {
		return fmt.Errorf("file not found: %s", p)
	}
	if !strings.HasSuffix(strings.ToLower(p), ".proto") {
		return fmt.Errorf("file not found: %s", p)
	}
	clean := filepath.ToSlash(filepath.Clean(p))
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("file not found: %s", p)
	}
	return nil
}

// safeDiskAccessor opens path but refuses symlinks and directories.
func safeDiskAccessor(path string) (io.ReadCloser, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("file not found: %s", path)
	}
	return os.Open(path) //nolint:gosec // validated by safeDiskResolver
}
