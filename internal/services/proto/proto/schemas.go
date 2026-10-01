// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"errors"
	"time"

	"github.com/altessa-s/go-atlas/core/encoding/hash"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	"google.golang.org/protobuf/reflect/protoreflect"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

func activeRevision(src *entities.ProtoSource) (string, error) {
	if !src.Enabled {
		return "", errs.ErrMappingSourceDisabled
	}
	switch src.SourceType {
	case entities.SourceTypeLocal:
		return LocalRevision, nil
	case entities.SourceTypeFiles:
		return FilesRevision, nil
	case entities.SourceTypeGit:
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
	sourceID, revision string,
	fds []protoreflect.FileDescriptor,
) (*entities.ProtoDescriptor, error) {
	descSet, err := s.serialize(fds)
	if err != nil {
		return nil, coreerrs.WrapOperation(err, "serialize descriptors")
	}
	d := entities.ProtoDescriptorNew(func(d *entities.ProtoDescriptor) {
		d.SourceID = sourceID
		d.Revision = revision
		d.DescriptorSet = descSet
		d.Fingerprint = hash.SHA256HexBytes(descSet)
		d.MessageTypes = s.extractTypes(fds)
		d.CompiledAt = time.Now().UnixMilli()
	})
	if err := s.descriptorsStorage.Save(ctx, d); err != nil {
		return nil, coreerrs.WrapOperation(err, "save descriptor")
	}
	s.registryCache.Invalidate(sourceID, revision)
	return d, nil
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
