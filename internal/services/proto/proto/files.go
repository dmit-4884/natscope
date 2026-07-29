// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/reporter"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"google.golang.org/protobuf/reflect/protoreflect"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// FilesTag is the synthetic descriptor tag for Files-type sources (one
// persisted descriptor per source).
const FilesTag = "manual"

// compileFiles runs protocompile against the file map plus include dirs.
// Compile errors return nil descriptors; third return is for real failures.
func compileFiles(
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
	diskResolver := &protocompile.SourceResolver{
		ImportPaths: includeDirs,
	}
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
) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	if sourceID != nil && *sourceID != "" {
		src, err := s.sourcesStorage.Get(ctx, *sourceID, false)
		if err != nil {
			return nil, nil, err
		}
		if src.SourceType != entities.SourceTypeFiles {
			return nil, nil, fmt.Errorf("%w: validate files is only available for files-type sources", errs.ErrInvalidRequest)
		}
		files = src.Files
		includeDirs = src.IncludeDirs
	}

	entries, preDiag := protoutils.ReadFilesFromPaths(files)
	dirDiag := protoutils.ValidateIncludeDirs(includeDirs)

	allDiag := append([]entities.CompileDiagnostic{}, preDiag...)
	allDiag = append(allDiag, dirDiag...)

	// Skip the compiler if pre-compile diagnostics already disqualify the input.
	if len(entries) == 0 || slices.Any(allDiag, func(d entities.CompileDiagnostic) bool {
		return d.Severity == entities.DiagnosticError
	}) {
		return nil, allDiag, nil
	}

	fds, compileDiag, err := compileFiles(ctx, entries, includeDirs)
	if err != nil {
		return nil, allDiag, err
	}
	allDiag = append(allDiag, compileDiag...)

	if len(fds) == 0 {
		return nil, allDiag, nil
	}

	return &entities.CompileResult{
		MessageTypes:    len(s.extractTypes(fds)),
		FileDescriptors: len(fds),
	}, allDiag, nil
}

// CompileFiles compiles a stored Files-type source and persists the descriptor
// under FilesTag. On error nothing persists; success still returns warnings.
func (s *Service) CompileFiles(
	ctx context.Context,
	sourceID string,
) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	unlock := s.compileLocks.lock(sourceID)
	defer unlock()

	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, nil, err
	}
	if src.SourceType != entities.SourceTypeFiles {
		return nil, nil, fmt.Errorf("%w: compile files is only available for files-type sources", errs.ErrInvalidRequest)
	}

	entries, preDiag := protoutils.ReadFilesFromPaths(src.Files)
	dirDiag := protoutils.ValidateIncludeDirs(src.IncludeDirs)
	allDiag := append([]entities.CompileDiagnostic{}, preDiag...)
	allDiag = append(allDiag, dirDiag...)

	if len(entries) == 0 || slices.Any(allDiag, func(d entities.CompileDiagnostic) bool {
		return d.Severity == entities.DiagnosticError
	}) {
		s.recordCompile(ctx, sourceID, false, summarizeDiagnostics(allDiag), 0, len(entries), allDiag, nil, "")
		return nil, allDiag, nil
	}

	fds, compileDiag, err := compileFiles(ctx, entries, src.IncludeDirs)
	if err != nil {
		s.recordCompile(ctx, sourceID, false, err.Error(), 0, len(entries), allDiag, nil, "")
		return nil, allDiag, err
	}
	allDiag = append(allDiag, compileDiag...)

	if len(fds) == 0 {
		s.recordCompile(ctx, sourceID, false, summarizeDiagnostics(allDiag), 0, len(entries), allDiag, nil, "")
		return nil, allDiag, nil
	}

	descSet, err := s.serialize(fds)
	if err != nil {
		s.recordCompile(ctx, sourceID, false, err.Error(), 0, len(fds), allDiag, nil, "")
		return nil, allDiag, coreerrs.WrapOperation(err, "serialize descriptors")
	}

	messageTypes := s.extractTypes(fds)
	s.recordCompile(ctx, sourceID, true, "", len(messageTypes), len(fds), allDiag, nil, "")

	descriptor := entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = sourceID
		d.Tag = FilesTag
		d.DescriptorSet = descSet
		d.MessageTypes = messageTypes
		d.CompiledAt = time.Now().UnixMilli()
	})
	if err := s.descriptorsStorage.Save(ctx, descriptor); err != nil {
		return nil, allDiag, coreerrs.WrapOperation(err, "save descriptor")
	}

	s.registryCache.Invalidate(sourceID, FilesTag)

	s.logger.InfoContext(ctx, "compiled files proto source",
		slog.String("source_id", sourceID),
		slog.Int("files", len(entries)),
		slog.Int("include_dirs", len(src.IncludeDirs)),
		slog.Int("message_types", len(messageTypes)),
		slog.Int("file_descriptors", len(fds)))

	s.notifyReload(ctx)

	return &entities.CompileResult{
		MessageTypes:    len(messageTypes),
		FileDescriptors: len(fds),
	}, allDiag, nil
}
