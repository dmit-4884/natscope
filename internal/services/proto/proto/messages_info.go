// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"context"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

// ListMessages returns all messages from active snapshots, each annotated
// with SourceID + SourceTag so the UI can disambiguate same-FQN sources.
func (s *Service) ListMessages(ctx context.Context) []entities.ProtoMessageInfo {
	snaps := s.activeSnapshots(ctx)
	if len(snaps) == 0 {
		return nil
	}

	messages := make([]entities.ProtoMessageInfo, 0, estimatedMessagesPerSnapshot)
	for _, snap := range snaps {
		for _, md := range snap.Messages {
			info := protoutils.Info(md)
			info.SourceID = snap.SourceID
			info.SourceTag = snap.Tag
			messages = append(messages, info)
		}
	}

	slices.SortFunc(messages, func(a, b entities.ProtoMessageInfo) int {
		if c := strings.Compare(a.FullName, b.FullName); c != 0 {
			return c
		}
		return strings.Compare(a.SourceID, b.SourceID)
	})

	return messages
}

// GetMessage returns message-type detail within a source's active snapshot.
func (s *Service) GetMessage(ctx context.Context, sourceID, messageType string) (*entities.ProtoMessageInfo, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	snap, err := s.snapshotForSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	md, ok := snap.Messages[messageType]
	if !ok {
		return nil, errs.ErrProtoMessageNotFound
	}
	info := protoutils.Info(md)
	info.SourceID = snap.SourceID
	info.SourceTag = snap.Tag
	return &info, nil
}

// GenerateExample generates an example JSON object for a message type within a
// source.
func (s *Service) GenerateExample(ctx context.Context, sourceID, messageType string) (map[string]interface{}, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}
	snap, err := s.snapshotForSource(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	md, ok := snap.Messages[messageType]
	if !ok {
		return nil, errs.ErrProtoMessageNotFound
	}
	return protoutils.Template(md, make(map[string]bool)), nil
}

// Stats returns aggregated statistics across all active source snapshots.
func (s *Service) Stats(ctx context.Context) *entities.ProtoStats {
	snaps := s.activeSnapshots(ctx)
	count := 0
	for _, snap := range snaps {
		count += len(snap.Messages)
	}
	return &entities.ProtoStats{MessagesCount: count}
}
