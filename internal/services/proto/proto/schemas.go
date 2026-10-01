// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/encoding/hash"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/reflect/protoreflect"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	stdslices "slices"
)

func activeRevision(src *entities.ProtoSource) (string, error) {
	if !src.Enabled {
		return "", errs.ErrMappingSourceDisabled
	}
	switch src.SourceType {
	case entities.SourceTypeLocal:
		return LocalRevision, nil
	case entities.SourceTypeUpload:
		if src.ActiveSchema == nil {
			return "", errs.ErrMappingSelectionMissing
		}
		return src.ActiveSchema.Revision, nil
	case entities.SourceTypeGit, entities.SourceTypeBSR:
		if src.SelectedRef == nil || src.SelectedRef.Revision == "" {
			return "", errs.ErrMappingSelectionMissing
		}
		return src.SelectedRef.Revision, nil
	default:
		return "", errs.ErrMappingSelectionMissing
	}
}

func (s *Service) storeSchema(
	ctx context.Context,
	src *entities.ProtoSource,
	revision string,
	fds []protoreflect.FileDescriptor,
) (*entities.ProtoDescriptor, error) {
	descSet, err := s.serialize(fds)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "serialize descriptors")
	}
	d, err := s.saveSchema(ctx, src.Id, revision, descSet, s.extractTypes(fds), slices.To(fds, protoreflect.FileDescriptor.Path),
		compileSettings(src))
	return d, err
}

func compileSettings(src *entities.ProtoSource) string {
	if len(src.ImportRoots) == 0 && len(src.ExcludePrefixes) == 0 {
		return ""
	}
	return strings.Join(src.ImportRoots, "\x00") + "\x01" + strings.Join(src.ExcludePrefixes, "\x00")
}

func (s *Service) saveSchema(
	ctx context.Context,
	sourceID, revision string,
	descSet []byte,
	messageTypes, targetFiles []string,
	settings string,
) (*entities.ProtoDescriptor, error) {
	d := entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = sourceID
		d.Revision = revision
		d.DescriptorSet = descSet
		d.Fingerprint = hash.SHA256HexBytes(descSet)
		d.MessageTypes = messageTypes
		d.TargetFiles = targetFiles
		d.CompiledAt = time.Now().UnixMilli()
		d.CompileSettings = settings
	})
	if err := s.descriptorsStorage.Save(ctx, d); err != nil {
		return nil, coreerrs.WrapOperation(err, "save descriptor")
	}
	s.registryCache.Invalidate(sourceID, revision)
	s.pruneRevisions(ctx, sourceID, revision)
	return d, nil
}

func (s *Service) pruneRevisions(ctx context.Context, sourceID, saved string) {
	stored, err := s.descriptorsStorage.ListBySource(ctx, sourceID)
	if err != nil || len(stored) <= maxStoredRevisions {
		return
	}
	keep := map[string]bool{saved: true}
	if src, getErr := s.sourcesStorage.Get(ctx, sourceID, false); getErr == nil && src.ActiveSchema != nil {
		keep[src.ActiveSchema.Revision] = true
	}
	pinned := map[string]bool{}
	if s.mappingsService != nil {
		all, listErr := s.mappingsService.GetAll(ctx)
		if listErr != nil {
			return
		}
		for _, m := range all {
			if m.SourceID == sourceID && m.PinnedFingerprint != nil {
				pinned[*m.PinnedFingerprint] = true
			}
		}
	}
	stdslices.SortFunc(stored, func(a, b *entities.ProtoDescriptor) int { return cmp.Compare(b.CompiledAt, a.CompiledAt) })
	for i, d := range stored {
		if i < maxStoredRevisions || keep[d.Revision] || pinned[d.Fingerprint] {
			continue
		}
		if delErr := s.descriptorsStorage.DeleteBySourceRevision(ctx, sourceID, d.Revision); delErr != nil {
			s.logger.WarnContext(ctx, "prune schema revision", slog.String("source_id", sourceID), slogx.Error(delErr))
			continue
		}
		//nolint:errcheck // a leftover file set only costs disk
		_ = s.fileSetsStorage.DeleteBySourceRevision(ctx, sourceID, d.Revision)
		s.registryCache.Invalidate(sourceID, d.Revision)
	}
}

func (s *Service) hasSchema(ctx context.Context, sourceID, revision string) (bool, error) {
	_, err := s.descriptorsStorage.GetBySourceRevision(ctx, sourceID, revision)
	if errors.Is(err, errs.ErrProtoDescriptorNotFound) {
		return false, nil
	}
	return err == nil, err
}

func schemaRevision(d *entities.ProtoDescriptor, active bool) *entities.SchemaRevision {
	return &entities.SchemaRevision{
		Revision:     d.Revision,
		Fingerprint:  d.Fingerprint,
		CompiledAt:   d.CompiledAt,
		MessageCount: int32(len(d.MessageTypes)), //nolint:gosec // bounded by descriptor size
		Active:       active,
	}
}
