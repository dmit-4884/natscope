// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// ListTypes returns the messages, enums and services of one source, or of every enabled source when sourceID is empty.
func (s *Service) ListTypes(ctx context.Context, sourceID string) ([]entities.SchemaType, error) {
	snaps, err := s.snapshotsFor(ctx, sourceID)
	if err != nil {
		return nil, err
	}

	var out []entities.SchemaType
	for _, snap := range snaps {
		for _, t := range snap.Schema.Summaries() {
			t.SourceID = snap.SourceID
			t.SourceRevision = snap.Revision
			t.Dependency = !slices.Contains(snap.Descriptor.TargetFiles, t.File)
			out = append(out, t)
		}
	}
	slices.SortFunc(out, func(a, b entities.SchemaType) int {
		return cmp.Or(strings.Compare(a.FullName, b.FullName), strings.Compare(a.SourceID, b.SourceID))
	})
	return out, nil
}

// DescribeType describes a message, enum or service of a source; reachable adds every type it leads to.
func (s *Service) DescribeType(ctx context.Context, sourceID, fullName string, reachable bool) (*entities.TypeDescription, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	snap, err := s.snapshotForSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	desc, ok := snap.Schema.Describe(fullName, reachable)
	if !ok {
		return nil, errs.ErrProtoTypeNotFound
	}
	return desc, nil
}

// GenerateExample generates an example JSON object for a message type within a
// source.
func (s *Service) GenerateExample(ctx context.Context, sourceID, messageType string) (any, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	snap, err := s.snapshotForSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	md, ok := snap.Schema.Message(messageType)
	if !ok {
		return nil, errs.ErrProtoMessageNotFound
	}
	return protoutils.Template(md), nil
}

// Stats returns aggregated statistics across all active source snapshots.
func (s *Service) Stats(ctx context.Context) *entities.ProtoStats {
	snaps := s.activeSnapshots(ctx)
	count := 0
	for _, snap := range snaps {
		count += len(snap.Schema.Messages)
	}
	return &entities.ProtoStats{MessagesCount: count}
}
