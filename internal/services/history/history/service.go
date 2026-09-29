// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"log/slog"
	"unicode/utf8"

	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
	historysvc "github.com/dmit-4884/natscope/internal/services/history"
	storage "github.com/dmit-4884/natscope/internal/storages/history"
)

// Service implements history.Service.
type Service struct {
	storage storage.Storage
	logger  *slog.Logger
}

// New creates a new history service.
func New(storage storage.Storage) *Service {
	return &Service{
		storage: storage,
		logger:  slog.Default().With(slogx.Module("service:history")),
	}
}

// Record creates a history entry; PayloadJSON is capped to
// HistoryPayloadPreviewBytes to keep the per-insert file rewrite from O(n²).
func (s *Service) Record(
	ctx context.Context,
	in *entities.PublishHistoryCreate,
) (*entities.PublishHistory, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	entry := converter.Convert(in, entities.PublishHistoryNew())

	entry.PayloadSize = in.PayloadSize
	if entry.PayloadSize == 0 {
		entry.PayloadSize = len(in.PayloadJSON)
	}
	if len(entry.PayloadJSON) > entities.HistoryPayloadPreviewBytes {
		entry.PayloadJSON = truncateAtRuneBoundary(entry.PayloadJSON, entities.HistoryPayloadPreviewBytes)
		entry.PayloadTruncated = true
	}

	if err := s.storage.Save(ctx, entry); err != nil {
		s.logger.ErrorContext(ctx, "failed to save history entry",
			slogx.Error(err))
		return nil, err
	}

	if pruned, err := s.storage.Prune(ctx, entities.HistoryMaxEntries, entities.HistoryPruneSlack); err != nil {
		s.logger.WarnContext(ctx, "failed to prune history", slogx.Error(err))
	} else if pruned > 0 {
		s.logger.DebugContext(ctx, "history pruned", slog.Int64("removed", pruned))
	}

	s.logger.DebugContext(ctx, "history entry recorded",
		slog.String("id", entry.Id),
		slog.String("subject", in.Subject),
		slog.Bool("success", in.Success))

	return entry, nil
}

// List returns history entries for a user with pagination.
func (s *Service) List(
	ctx context.Context,
	in *entities.PublishHistoryList,
) (*entities.List[entities.PublishHistories], error) {
	return s.storage.List(ctx, in)
}

// truncateAtRuneBoundary cuts s to at most maxBytes without splitting a UTF-8 rune.
func truncateAtRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// Compile-time interface check.
var _ historysvc.Service = (*Service)(nil)
