// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

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
// URL lookup, stream publishing and core requests. The roles are injected
// separately so fx can resolve each, then embedded here for internal use.
type natsDeps struct {
	natssvc.ConnectionManager
	natssvc.Publisher
	natssvc.Requester
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
	requester natssvc.Requester,
	protoService protosvc.Codec,
	historyService historysvc.Service,
	settingsService settingssvc.Service,
) *Service {
	return &Service{
		natsService:     natsDeps{connManager, publisher, requester},
		protoService:    protoService,
		historyService:  historyService,
		settingsService: settingsService,
		logger:          slog.Default().With(slogx.Module("service:publish")),
	}
}

// Publish runs the full pipeline; every user-actionable failure becomes a PublishResult.Error (see package doc).
// A malformed subject or header name fails first with a transport error.
func (s *Service) Publish(ctx context.Context, in *entities.PublishRequest) (*entities.PublishResult, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := natsutil.ValidateLiteralSubject(in.Subject); err != nil {
		return nil, err
	}
	if err := natsutil.ValidateHeaderNames(in.Headers); err != nil {
		return nil, err
	}

	data, encErr := s.resolvePayload(ctx, in)
	if encErr != nil {
		if !in.Core {
			s.recordHistory(ctx, in, nil, len(data), encErr)
		}
		return softFailure(*encErr), nil
	}

	pubCtx, cancel := corecontext.ApplyTimeout(ctx, s.publishTimeout(ctx))
	defer cancel()

	if in.Core {
		return s.publishCore(pubCtx, in, data)
	}

	ack, err := s.natsService.PublishToStream(pubCtx, in.ConnectionID, in.Subject, data, in.Headers)
	if err != nil {
		if errors.Is(err, errs.ErrSavedConnectionNotFound) {
			return nil, err
		}
		failure := publishFailure(err)
		s.recordHistory(ctx, in, nil, len(data), failure.Error)
		return failure, nil
	}

	s.recordHistory(ctx, in, ack, len(data), nil)
	result := &entities.PublishResult{
		Stream:    ack.Stream,
		Sequence:  ack.Sequence,
		Duplicate: ack.Duplicate,
	}
	if ack.Value != "" {
		result.CounterValue = &ack.Value
	}
	return result, nil
}

func (s *Service) publishCore(ctx context.Context, in *entities.PublishRequest, data []byte) (*entities.PublishResult, error) {
	if err := s.natsService.Publish(ctx, in.ConnectionID, in.Subject, data, in.Headers); err != nil {
		if errors.Is(err, errs.ErrSavedConnectionNotFound) {
			return nil, err
		}
		return publishFailure(err), nil
	}
	return &entities.PublishResult{}, nil
}

func publishFailure(err error) *entities.PublishResult {
	result := softFailure("Failed to publish message: " + err.Error())
	if permErr, ok := errors.AsType[*errs.NATSPermissionError](err); ok {
		result.Access = &entities.AccessCheck{Status: entities.AccessDenied, Operation: permErr.Operation, Subject: permErr.Subject}
	}
	return result
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
		Framing:     in.Framing,
	}
	if in.SchemaFingerprint != nil {
		req.Fingerprint = *in.SchemaFingerprint
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

// recordHistory persists one publish attempt: ack is nil and errMsg set when it
// failed. History errors are best-effort since the publish has already settled.
func (s *Service) recordHistory(
	ctx context.Context,
	in *entities.PublishRequest,
	ack *entities.PubAck,
	payloadSize int,
	errMsg *string,
) {
	encoding := entities.EncodingTypeJSON
	var msgType string
	switch {
	case in.MessageType != nil && *in.MessageType != "":
		msgType = *in.MessageType
		encoding = entities.EncodingTypeProtobuf
	case !json.Valid([]byte(in.Data)):
		encoding = entities.EncodingTypeText
	}

	connURL, urlErr := s.natsService.GetConnectionURL(ctx, in.ConnectionID)
	if urlErr != nil && errors.Is(urlErr, errs.ErrSavedConnectionNotFound) {
		// Skip history for a connection id that doesn't exist.
		return
	}

	create := &entities.PublishHistoryCreate{
		ConnectionID:   &in.ConnectionID,
		ConnectionURL:  connURL,
		Subject:        in.Subject,
		SubjectPattern: in.SubjectPattern,
		EncodingType:   encoding,
		MessageType:    msgType,
		PayloadJSON:    in.Data,
		PayloadSize:    payloadSize,
		Success:        errMsg == nil,
		Headers:        in.Headers,
		Error:          errMsg,
	}
	if ack != nil {
		create.Stream = ack.Stream
		create.Duplicate = ack.Duplicate
		if ack.Sequence > 0 {
			create.Sequence = &ack.Sequence
		}
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
