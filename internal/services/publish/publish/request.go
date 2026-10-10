// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package publish

import (
	"cmp"
	"context"
	"time"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsutil"

	corecontext "github.com/altessa-s/go-atlas/core/context"
)

// defaultRequestTimeout applies when a request sets no timeout.
const defaultRequestTimeout = 5 * time.Second

// Request validates and encodes the payload like Publish, then sends it as a
// core NATS request; the request's own timeout always bounds the wait.
func (s *Service) Request(ctx context.Context, in *entities.RequestMessage) (*entities.Reply, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := natsutil.ValidateLiteralSubject(in.Subject); err != nil {
		return nil, err
	}
	if err := natsutil.ValidateHeaderNames(in.Headers); err != nil {
		return nil, err
	}
	if err := s.natsService.EnsureWritable(ctx, in.ConnectionID); err != nil && refused(err) {
		return nil, err
	}

	data, encErr := s.resolvePayload(ctx, &in.PublishRequest)
	if encErr != nil {
		return nil, &errs.ProtoEncodeError{Description: *encErr}
	}

	reqCtx, cancel := corecontext.WithMaxTimeout(ctx, cmp.Or(in.Timeout, defaultRequestTimeout))
	defer cancel()

	return s.natsService.Request(reqCtx, in.ConnectionID, in.Subject, data, in.Headers)
}
