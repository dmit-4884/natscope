// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package history

import (
	"context"
	"log/slog"

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

	// PayloadSize keeps the original byte count; PayloadJSON is the preview.
	entry.PayloadSize = len(in.PayloadJSON)
	if len(entry.PayloadJSON) > entities.HistoryPayloadPreviewBytes {
		entry.PayloadJSON = entry.PayloadJSON[:entities.HistoryPayloadPreviewBytes]
		entry.PayloadTruncated = true
	}

	if err := s.storage.Save(ctx, entry); err != nil {
		s.logger.ErrorContext(ctx, "failed to save history entry",
			slogx.Error(err))
		return nil, err
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

// Compile-time interface check.
var _ historysvc.Service = (*Service)(nil)
