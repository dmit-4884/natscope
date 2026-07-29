// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// ListStreams returns all JetStream streams.
func (s *Service) ListStreams(ctx context.Context, connectionID string) ([]entities.StreamInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.ListStreams(ctx)
}

// GetStreamInfo returns detailed information about a specific stream.
func (s *Service) GetStreamInfo(ctx context.Context, connectionID string, streamName string) (*entities.StreamInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetStreamInfo(ctx, streamName)
}

// GetStreamConsumers returns detailed consumer information for a stream.
func (s *Service) GetStreamConsumers(
	ctx context.Context,
	connectionID string,
	streamName string,
) ([]entities.ConsumerInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetStreamConsumers(ctx, streamName)
}

// GetAllStreamsStats returns statistics for all streams in the connection.
func (s *Service) GetAllStreamsStats(ctx context.Context, connectionID string) ([]entities.StreamStats, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetAllStreamsStats(ctx)
}

// GetStreamStats returns detailed statistics for a specific stream.
func (s *Service) GetStreamStats(ctx context.Context, connectionID string, streamName string) (*entities.StreamStats, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetStreamStats(ctx, streamName)
}

// CreateStream creates a new JetStream stream with the given configuration.
func (s *Service) CreateStream(
	ctx context.Context,
	connectionID string,
	config entities.StreamCreateRequest,
) (*entities.StreamInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.CreateStream(ctx, config)
}

// UpdateStream updates an existing JetStream stream configuration.
func (s *Service) UpdateStream(
	ctx context.Context,
	connectionID string,
	name string,
	config entities.StreamUpdateRequest,
) (*entities.StreamInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.UpdateStream(ctx, name, config)
}

// DeleteStream deletes a JetStream stream and all its data.
func (s *Service) DeleteStream(ctx context.Context, connectionID string, name string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteStream(ctx, name)
}

// PurgeStream removes messages from a stream based on the purge request
// options.
func (s *Service) PurgeStream(
	ctx context.Context,
	connectionID string,
	name string,
	req entities.StreamPurgeRequest,
) (uint64, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return 0, err
	}
	return c.PurgeStream(ctx, name, req)
}

// DeleteMessage deletes one message by sequence; secure=true overwrites data
// first (irreversible erase).
func (s *Service) DeleteMessage(
	ctx context.Context,
	connectionID string,
	streamName string,
	sequence uint64,
	secure bool,
) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.DeleteMessage(ctx, streamName, sequence, secure)
}

// SealStream seals a stream, making it read-only, and returns its updated info.
func (s *Service) SealStream(ctx context.Context, connectionID string, name string) (*entities.StreamInfo, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.SealStream(ctx, name)
}
