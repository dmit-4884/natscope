// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// GetConsumersOverview returns every readable consumer across all streams, with the streams.
func (s *Service) GetConsumersOverview(ctx context.Context, connectionID string) (*entities.ConsumersOverview, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetConsumersOverview(ctx)
}

// CreateConsumer creates a new consumer on a stream.
func (s *Service) CreateConsumer(
	ctx context.Context,
	connectionID, streamName string,
	config entities.ConsumerCreateRequest,
) (*entities.ConsumerInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.CreateConsumer(ctx, streamName, config)
}

// UpdateConsumer updates an existing consumer configuration.
func (s *Service) UpdateConsumer(
	ctx context.Context,
	connectionID, streamName, consumerName string,
	config entities.ConsumerUpdateRequest,
) (*entities.ConsumerInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.UpdateConsumer(ctx, streamName, consumerName, config)
}

// DeleteConsumer deletes a consumer from a stream.
func (s *Service) DeleteConsumer(
	ctx context.Context,
	connectionID string,
	streamName string,
	consumerName string,
) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteConsumer(ctx, streamName, consumerName)
}

// PauseConsumer pauses a consumer until the supplied RFC3339 timestamp.
func (s *Service) PauseConsumer(
	ctx context.Context,
	connectionID, streamName, consumerName string,
	pauseUntil string,
) (*entities.ConsumerPauseResponse, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.PauseConsumer(ctx, streamName, consumerName, pauseUntil)
}

// ResumeConsumer resumes a paused consumer and returns the server's post-resume state.
func (s *Service) ResumeConsumer(
	ctx context.Context,
	connectionID string,
	streamName string,
	consumerName string,
) (*entities.ConsumerPauseResponse, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ResumeConsumer(ctx, streamName, consumerName)
}

// ResetConsumer resets a consumer's delivery state, optionally to a stream sequence.
func (s *Service) ResetConsumer(
	ctx context.Context,
	connectionID string,
	streamName string,
	consumerName string,
	sequence *uint64,
) (*entities.ConsumerResetResponse, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ResetConsumer(ctx, streamName, consumerName, sequence)
}

// UnpinConsumer releases the pinned client of a consumer priority group.
func (s *Service) UnpinConsumer(
	ctx context.Context,
	connectionID string,
	streamName string,
	consumerName string,
	group string,
) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.UnpinConsumer(ctx, streamName, consumerName, group)
}
