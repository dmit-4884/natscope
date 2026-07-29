// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/errors"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// ValidateLocalPath probes a filesystem root, counting only includable
// .proto files (skips .git/node_modules); error means a genuine internal failure.
func (s *Service) ValidateLocalPath(_ context.Context, dirPath string) (*entities.LocalPathValidation, error) {
	failure := func(msg string) (*entities.LocalPathValidation, error) {
		return &entities.LocalPathValidation{Valid: false, Error: &msg}, nil
	}

	info, err := os.Stat(dirPath)
	if err != nil {
		return failure("path not accessible: " + err.Error())
	}
	if !info.IsDir() {
		return failure("path is not a directory")
	}

	walk, err := protoutils.WalkProtoTree(dirPath, protoutils.WalkOptions{})
	if err != nil {
		return failure("path not accessible: " + err.Error())
	}
	if len(walk.Files) == 0 {
		return failure("no .proto files found in directory")
	}
	return &entities.LocalPathValidation{Valid: true, ProtoFileCount: len(walk.Files)}, nil
}

// CompileLocal compiles a local dir source; compile diagnostics return via
// the slice with nil error—error is reserved for misconfiguration or internal failures.
func (s *Service) CompileLocal(
	ctx context.Context,
	sourceID string,
) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	source, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, nil, err
	}
	if source.SourceType != entities.SourceTypeLocal {
		return nil, nil, fmt.Errorf("%w: compile local is only available for local directory sources", errs.ErrInvalidRequest)
	}
	if source.LocalPath == nil || *source.LocalPath == "" {
		return nil, nil, fmt.Errorf("%w: local path not configured for source", errs.ErrInvalidRequest)
	}
	return s.compileLocalLocked(ctx, source)
}

// compileLocalLocked is the shared local-compile pipeline for manual RPC and
// file-watcher; reads the dir inside the per-source lock to avoid stale snapshots.
func (s *Service) compileLocalLocked(
	ctx context.Context,
	source *entities.ProtoSource,
) (*entities.CompileResult, []entities.CompileDiagnostic, error) {
	unlock := s.compileLocks.lock(source.Id)
	defer unlock()

	walk, err := protoutils.WalkProtoTree(*source.LocalPath, protoutils.WalkOptions{
		ExcludePrefixes: source.ExcludePrefixes,
		CollectConfigs:  true,
	})
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, 0, nil, nil, "")
		return nil, nil, fmt.Errorf("%w: failed to read local directory: %s", errs.ErrInvalidRequest, err.Error())
	}

	diags := skippedToDiagnostics(walk.Skipped)
	out, err := s.compile(ctx, source, walk.Files, walk.Configs, *source.LocalPath)
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, len(walk.Files), diags, nil, "")
		return nil, diags, err
	}
	diags = append(diags, out.Diags...)

	if out.HasErrors() {
		s.recordCompile(
			ctx, source.Id, false, summarizeDiagnostics(diags),
			0, len(walk.Files), diags, out.Roots, string(out.Origin),
		)
		return nil, diags, nil
	}

	descSet, err := s.serialize(out.FDS)
	if err != nil {
		s.recordCompile(ctx, source.Id, false, err.Error(), 0, len(out.FDS), diags, out.Roots, string(out.Origin))
		return nil, diags, errors.WrapOperation(err, "serialize descriptors")
	}
	messageTypes := s.extractTypes(out.FDS)
	s.recordCompile(ctx, source.Id, true, "", len(messageTypes), len(out.FDS), diags, out.Roots, string(out.Origin))

	descriptor := entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = source.Id
		d.Tag = LocalTag
		d.DescriptorSet = descSet
		d.MessageTypes = messageTypes
		d.CompiledAt = time.Now().UnixMilli()
	})
	if err := s.descriptorsStorage.Save(ctx, descriptor); err != nil {
		return nil, diags, errors.WrapOperation(err, "save descriptor")
	}
	s.registryCache.Invalidate(source.Id, LocalTag)
	s.notifyReload(ctx)

	return &entities.CompileResult{
		MessageTypes:    len(messageTypes),
		FileDescriptors: len(out.FDS),
	}, diags, nil
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
