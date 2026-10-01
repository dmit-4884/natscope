// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"fmt"
	"os"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// localPathInvalidMessage covers every ValidateLocalPath failure, so the RPC can't probe arbitrary paths.
const localPathInvalidMessage = "no .proto files found at this path"

// ValidateLocalPath probes a filesystem root, counting only includable
// .proto files (skips .git/node_modules); error means a genuine internal failure.
func (s *Service) ValidateLocalPath(_ context.Context, dirPath string) (*entities.LocalPathValidation, error) {
	invalid := func() (*entities.LocalPathValidation, error) {
		msg := localPathInvalidMessage
		return &entities.LocalPathValidation{Valid: false, Error: &msg}, nil
	}

	info, err := os.Stat(dirPath)
	if err != nil || !info.IsDir() {
		return invalid()
	}

	walk, err := protoutils.WalkProtoTree(dirPath, protoutils.WalkOptions{})
	if err != nil || len(walk.Files) == 0 {
		return invalid()
	}
	return &entities.LocalPathValidation{Valid: true, ProtoFileCount: len(walk.Files)}, nil
}

func (s *Service) compileLocal(ctx context.Context, source *entities.ProtoSource) (*entities.CompileOutcome, error) {
	if source.LocalPath == nil || *source.LocalPath == "" {
		return nil, fmt.Errorf("%w: local path not configured for source", errs.ErrInvalidRequest)
	}

	unlock := s.compileLocks.lock(source.Id)
	defer unlock()

	walk, err := protoutils.WalkProtoTree(*source.LocalPath, protoutils.WalkOptions{
		ExcludePrefixes: source.ExcludePrefixes,
		CollectConfigs:  true,
	})
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, 0, nil, nil, "", nil)
		return nil, fmt.Errorf("%w: failed to read local directory: %s", errs.ErrInvalidRequest, err.Error())
	}

	diags := skippedToDiagnostics(walk.Skipped)
	out, err := s.compile(ctx, source, walk.Files, walk.Configs, *source.LocalPath)
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, len(walk.Files), diags, nil, "", nil)
		return nil, err
	}
	diags = append(diags, out.Diags...)

	if out.HasErrors() {
		s.recordCompile(ctx, source.Id, false, summarizeDiagnostics(diags),
			0, len(walk.Files), diags, out.Roots, string(out.Origin), nil)
		return &entities.CompileOutcome{Diagnostics: diags}, nil
	}

	d, err := s.storeSchema(ctx, source.Id, LocalRevision, out.FDS)
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, len(out.FDS), diags, out.Roots, string(out.Origin), nil)
		return nil, err
	}
	s.recordCompile(ctx, source.Id, true, "", len(d.MessageTypes), len(out.FDS),
		diags, out.Roots, string(out.Origin), schemaRevision(d, true))
	s.notifyReload(ctx)

	return &entities.CompileOutcome{
		Valid:           true,
		MessageTypes:    len(d.MessageTypes),
		FileDescriptors: len(out.FDS),
		Diagnostics:     diags,
	}, nil
}

// skippedToDiagnostics surfaces walk skips as INFO diagnostics so a missing
// file is never a mystery.
func skippedToDiagnostics(skipped []protoutils.SkippedFile) []entities.CompileDiagnostic {
	return slices.To(skipped, func(sk protoutils.SkippedFile) entities.CompileDiagnostic {
		return entities.CompileDiagnostic{
			Severity: entities.DiagnosticInfo,
			File:     sk.Path,
			Message:  "skipped during walk: " + sk.Reason,
		}
	})
}
