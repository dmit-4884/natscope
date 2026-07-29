// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"log/slog"
	"time"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"

	corecontext "github.com/altessa-s/go-atlas/core/context"
	slogx "github.com/altessa-s/go-atlas/observability/slog"
	historysvc "github.com/dmit-4884/natscope/internal/services/history"
	natssvc "github.com/dmit-4884/natscope/internal/services/nats"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
	publishsvc "github.com/dmit-4884/natscope/internal/services/publish"
	settingssvc "github.com/dmit-4884/natscope/internal/services/settings"
)

// defaultPublishTimeout applies when Settings.Publish.PublishTimeoutSec is
// unset.
const defaultPublishTimeout = 10 * time.Second

// natsDeps bundles the narrow NATS roles the publish service needs: connection
// URL lookup plus stream publishing. The roles are injected separately so fx can
// resolve each, then embedded here for internal use.
type natsDeps struct {
	natssvc.ConnectionManager
	natssvc.Publisher
}

// Service implements publishsvc.Service.
type Service struct {
	natsService     natsDeps
	protoService    protosvc.Codec
	historyService  historysvc.Service
	settingsService settingssvc.Service
	logger          *slog.Logger
}

// New creates a new publish coordinator service.
func New(
	connManager natssvc.ConnectionManager,
	publisher natssvc.Publisher,
	protoService protosvc.Codec,
	historyService historysvc.Service,
	settingsService settingssvc.Service,
) *Service {
	return &Service{
		natsService:     natsDeps{connManager, publisher},
		protoService:    protoService,
		historyService:  historyService,
		settingsService: settingsService,
		logger:          slog.Default().With(slogx.Module("service:publish")),
	}
}

// Publish runs the full pipeline; every user-actionable failure becomes a
// PublishResult.Error (see package doc).
func (s *Service) Publish(ctx context.Context, in *entities.PublishRequest) (*entities.PublishResult, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	data, encErr := s.resolvePayload(ctx, in)
	if encErr != nil {
		return softFailure(*encErr), nil
	}

	pubCtx, cancel := corecontext.ApplyTimeout(ctx, s.publishTimeout(ctx))
	defer cancel()

	ack, err := s.natsService.PublishToStream(pubCtx, in.ConnectionID, in.Subject, data, in.Headers)
	if err != nil {
		errMsg := "Failed to publish message: " + err.Error()
		s.recordHistory(ctx, in, "", 0, len(data), false, &errMsg)
		return softFailure(errMsg), nil
	}

	s.recordHistory(ctx, in, ack.Stream, ack.Sequence, len(data), true, nil)
	return &entities.PublishResult{
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Duplicate: ack.Duplicate,
	}, nil
}

// resolvePayload returns the bytes to publish: Data proto-encoded when
// MessageType is set, else verbatim; non-nil pointer means encode failed.
func (s *Service) resolvePayload(ctx context.Context, in *entities.PublishRequest) ([]byte, *string) {
	if in.MessageType == nil || *in.MessageType == "" {
		return []byte(in.Data), nil
	}
	if in.SourceID == nil || *in.SourceID == "" {
		msg := "source_id is required when message_type is set"
		return nil, &msg
	}

	req := entities.CodecRequest{
		JSON:        []byte(in.Data),
		SourceID:    *in.SourceID,
		MessageType: *in.MessageType,
	}
	if in.SourceTag != nil {
		req.Tag = *in.SourceTag
	}

	encoded, err := s.protoService.EncodeRaw(ctx, req)
	if err != nil {
		msg := err.Error()
		return nil, &msg
	}
	return encoded, nil
}

// publishTimeout reads PublishTimeoutSec, falling back to defaultPublishTimeout
// when missing or unreadable.
func (s *Service) publishTimeout(ctx context.Context) time.Duration {
	cfg, err := s.settingsService.Get(ctx)
	if err != nil || cfg == nil || cfg.Publish == nil || cfg.Publish.PublishTimeoutSec == nil {
		return defaultPublishTimeout
	}
	return time.Duration(*cfg.Publish.PublishTimeoutSec) * time.Second
}

// recordHistory persists one publish attempt; history errors are best-effort
// since the publish has already settled.
func (s *Service) recordHistory(
	ctx context.Context,
	in *entities.PublishRequest,
	stream string,
	sequence uint64,
	payloadSize int,
	success bool,
	errMsg *string,
) {
	encoding := entities.EncodingTypeJSON
	var msgType string
	if in.MessageType != nil && *in.MessageType != "" {
		msgType = *in.MessageType
		encoding = entities.EncodingTypeProtobuf
	}

	connURL, _ := s.natsService.GetConnectionURL(ctx, in.ConnectionID) //nolint:errcheck // history-only enrichment

	create := &entities.PublishHistoryCreate{
		ConnectionID:   &in.ConnectionID,
		ConnectionURL:  connURL,
		Stream:         stream,
		Subject:        in.Subject,
		SubjectPattern: in.SubjectPattern,
		EncodingType:   encoding,
		MessageType:    msgType,
		PayloadJSON:    in.Data,
		PayloadSize:    payloadSize,
		Success:        success,
		Error:          errMsg,
	}
	if sequence > 0 {
		create.Sequence = &sequence
	}

	if _, err := s.historyService.Record(ctx, create); err != nil {
		s.logger.ErrorContext(ctx, "failed to record publish history",
			slogx.Error(err),
			slog.String("subject", in.Subject))
	}
}

func softFailure(msg string) *entities.PublishResult {
	return &entities.PublishResult{Error: &msg}
}

var _ publishsvc.Service = (*Service)(nil)
