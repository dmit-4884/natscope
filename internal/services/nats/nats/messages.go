// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// GetMessages fetches a page of messages with the given options.
func (s *Service) GetMessages(
	ctx context.Context,
	connectionID string,
	streamName string,
	opts entities.GetMessagesOptions,
) (*entities.MessagesResponse, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetMessages(ctx, streamName, opts)
}

// GetMessage fetches a single message by sequence number.
func (s *Service) GetMessage(
	ctx context.Context,
	connectionID string,
	streamName string,
	sequence uint64,
) (*entities.Message, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetMessage(ctx, streamName, sequence)
}

// PublishToStream publishes a message to a JetStream stream.
func (s *Service) PublishToStream(
	ctx context.Context,
	connectionID string,
	subject string,
	data []byte,
	headers map[string]string,
) (*entities.PubAck, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.PublishToStream(ctx, subject, data, headers)
}
