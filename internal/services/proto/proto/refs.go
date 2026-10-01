// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// ListRefs returns the tags and branches of a Git source, or the labels of a BSR source.
func (s *Service) ListRefs(ctx context.Context, sourceID string) ([]entities.ProtoRef, error) {
	src, err := s.refSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if src.SourceType == entities.SourceTypeBSR {
		return s.bsrRefs(ctx, src)
	}
	return s.gitFetcher.ListRefs(ctx, src.AuthenticatedURL())
}

// SelectRef points a Git or BSR source at a ref; compile errors come back in the outcome.
func (s *Service) SelectRef(ctx context.Context, sourceID, ref string) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	src, err := s.refSource(ctx, sourceID)
	if err != nil {
		return nil, nil, err
	}
	resolved, err := s.resolveRef(ctx, src, ref)
	if err != nil {
		return nil, nil, err
	}
	return s.activateRef(ctx, src.Id, resolved)
}

// RefreshSource rebuilds the active schema: re-resolves the selected ref or recompiles the files.
func (s *Service) RefreshSource(ctx context.Context, sourceID string) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, nil, err
	}

	var outcome *entities.CompileOutcome
	switch src.SourceType {
	case entities.SourceTypeGit, entities.SourceTypeBSR:
		if src.SelectedRef == nil {
			return nil, nil, fmt.Errorf("%w: select a ref first", errs.ErrInvalidRequest)
		}
		resolved, resolveErr := s.resolveRef(ctx, src, src.SelectedRef.Name)
		if resolveErr != nil {
			return nil, nil, resolveErr
		}
		return s.refreshRef(ctx, src, resolved)
	case entities.SourceTypeLocal:
		outcome, err = s.compileLocal(ctx, src)
	case entities.SourceTypeUpload:
		outcome, err = s.compileUpload(ctx, src)
	default:
		return nil, nil, fmt.Errorf("%w: unsupported source type %q", errs.ErrInvalidRequest, src.SourceType)
	}
	if err != nil {
		return nil, outcome, err
	}
	updated, err := s.GetSource(ctx, sourceID)
	return updated, outcome, err
}

// ListRevisions returns the stored schemas of a source, newest first.
func (s *Service) ListRevisions(ctx context.Context, sourceID string) ([]entities.SchemaRevision, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, err
	}
	stored, err := s.descriptorsStorage.ListBySource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	active, _ := activeRevision(src) //nolint:errcheck // no active revision marks nothing active
	out := make([]entities.SchemaRevision, 0, len(stored))
	for _, d := range stored {
		out = append(out, *schemaRevision(d, d.Revision == active))
	}
	slices.SortFunc(out, func(a, b entities.SchemaRevision) int { return cmp.Compare(b.CompiledAt, a.CompiledAt) })
	return out, nil
}

func (s *Service) refSource(ctx context.Context, sourceID string) (*entities.ProtoSource, error) {
	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, err
	}
	if src.SourceType != entities.SourceTypeGit && src.SourceType != entities.SourceTypeBSR {
		return nil, fmt.Errorf("%w: refs exist only for Git and BSR sources", errs.ErrInvalidRequest)
	}
	return src, nil
}

func (s *Service) resolveRef(ctx context.Context, src *entities.ProtoSource, ref string) (entities.ProtoRef, error) {
	if src.SourceType == entities.SourceTypeBSR {
		return s.resolveBSRRef(ctx, src, ref)
	}
	return s.gitFetcher.ResolveRef(ctx, src.AuthenticatedURL(), ref)
}

func (s *Service) activateRef(
	ctx context.Context,
	sourceID string,
	ref entities.ProtoRef,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	return s.switchRef(ctx, sourceID, ref, "", false)
}

func (s *Service) refreshRef(
	ctx context.Context,
	src *entities.ProtoSource,
	ref entities.ProtoRef,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	return s.switchRef(ctx, src.Id, ref, src.SelectedRef.Name, src.SourceType == entities.SourceTypeGit)
}

func (s *Service) switchRef(
	ctx context.Context,
	sourceID string,
	ref entities.ProtoRef,
	expectedRef string,
	rebuild bool,
) (*entities.ProtoSource, *entities.CompileOutcome, error) {
	unlock := s.compileLocks.lock(sourceID)
	defer unlock()

	src, err := s.sourcesStorage.Get(ctx, sourceID, false)
	if err != nil {
		return nil, nil, err
	}
	if expectedRef != "" && (src.SelectedRef == nil || src.SelectedRef.Name != expectedRef) {
		current, getErr := s.GetSource(ctx, sourceID)
		return current, &entities.CompileOutcome{Valid: true}, getErr
	}

	d, err := s.descriptorsStorage.GetBySourceRevision(ctx, sourceID, ref.Revision)
	if rebuild && err == nil && d.CompileSettings != compileSettings(src) {
		err = errs.ErrProtoDescriptorNotFound
	}
	outcome := &entities.CompileOutcome{Valid: true}
	switch {
	case err == nil:
		outcome.MessageTypes = len(d.MessageTypes)
	case errors.Is(err, errs.ErrProtoDescriptorNotFound):
		if src.SourceType == entities.SourceTypeBSR {
			d, outcome, err = s.storeBSRRevision(ctx, src, ref)
		} else {
			d, outcome, err = s.compileGitRevision(ctx, src, ref)
		}
		if err != nil {
			return nil, outcome, err
		}
		if !outcome.Valid {
			current, getErr := s.GetSource(ctx, sourceID)
			return current, outcome, getErr
		}
		ref.Revision = d.Revision
	default:
		return nil, nil, err
	}

	s.patchSource(ctx, sourceID, func(src *entities.ProtoSource) {
		src.SelectedRef = &ref
		src.ActiveSchema = schemaRevision(d, true)
	})
	s.registryCache.InvalidateSource(sourceID)
	s.notifyReload(ctx)

	s.logger.InfoContext(ctx, "selected ref",
		slog.String("source_id", sourceID),
		slog.String("ref", ref.Name),
		slog.String("revision", ref.Revision))

	updated, err := s.GetSource(ctx, sourceID)
	return updated, outcome, err
}

func (s *Service) compileGitRevision(
	ctx context.Context,
	src *entities.ProtoSource,
	ref entities.ProtoRef,
) (*entities.ProtoDescriptor, *entities.CompileOutcome, error) {
	fileSet, err := s.fileSetsStorage.GetBySourceRevision(ctx, src.Id, ref.Revision)
	fetched := errors.Is(err, errs.ErrProtoFileSetNotFound)
	switch {
	case fetched:
		res, fetchErr := s.gitFetcher.Fetch(ctx, src.AuthenticatedURL(), ref)
		if fetchErr != nil {
			s.recordCompile(ctx, src.Id, false, fetchErr.Error(), 0, 0, nil, nil, "", nil)
			return nil, nil, fetchErr
		}
		fileSet = entities.ProtoFileSetNew(func(fs *entities.ProtoFileSet) {
			fs.SourceID = src.Id
			fs.Revision = res.Revision
			fs.Files = res.Files
			fs.Configs = res.Configs
			fs.FetchedAt = time.Now().UTC()
		})
	case err != nil:
		return nil, nil, err
	}

	out, err := s.compile(ctx, src, fileSet.Files, fileSet.Configs, "")
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, len(fileSet.Files), nil, nil, "", nil)
		return nil, nil, coreerrs.WrapOperation(err, "compile proto files")
	}
	if out.HasErrors() {
		s.recordCompile(ctx, src.Id, false, summarizeDiagnostics(out.Diags), 0, len(fileSet.Files),
			out.Diags, out.Roots, string(out.Origin), nil)
		return nil, &entities.CompileOutcome{Diagnostics: out.Diags}, nil
	}

	if fetched {
		if saveErr := s.fileSetsStorage.Save(ctx, fileSet); saveErr != nil {
			return nil, nil, coreerrs.WrapOperation(saveErr, "save proto files")
		}
	}
	d, err := s.storeSchema(ctx, src, fileSet.Revision, out.FDS)
	if err != nil {
		s.recordCompile(ctx, src.Id, false, err.Error(), 0, len(out.FDS), out.Diags, out.Roots, string(out.Origin), nil)
		return nil, nil, err
	}
	s.recordCompile(ctx, src.Id, true, "", len(d.MessageTypes), len(out.FDS), out.Diags, out.Roots, string(out.Origin), nil)

	return d, &entities.CompileOutcome{
		Valid:           true,
		MessageTypes:    len(d.MessageTypes),
		FileDescriptors: len(out.FDS),
		Diagnostics:     out.Diags,
	}, nil
}
