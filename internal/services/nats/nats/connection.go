// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
)

// GetConnectionURL returns the NATS server URL for the given connection ID.
func (s *Service) GetConnectionURL(ctx context.Context, connectionID string) (string, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return "", err
	}
	return c.URL(), nil
}

// GetConnectionHealth returns health status for a specific connection.
func (s *Service) GetConnectionHealth(ctx context.Context, connectionID string) (*entities.ConnectionHealth, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.Health(ctx)
}

// GetStreamSubjects returns the subject patterns for a specific stream.
func (s *Service) GetStreamSubjects(ctx context.Context, connectionID string, streamName string) ([]string, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetStreamSubjects(ctx, streamName)
}

// Subscribe creates a Core NATS subscription for the given subject.
func (s *Service) Subscribe(
	ctx context.Context,
	connectionID string,
	subject string,
	handler entities.MessageHandler,
) (entities.Subscription, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.Subscribe(ctx, subject, handler)
}

// SubscribeJetStream creates a JetStream ordered consumer subscription for live
// messages.
func (s *Service) SubscribeJetStream(
	ctx context.Context,
	connectionID, streamName, subject, deliverPolicy string,
	handler entities.MessageHandler,
) (entities.Subscription, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.SubscribeJetStream(ctx, streamName, subject, deliverPolicy, handler)
}

// TestConnection tests a NATS connection without saving it, dialing ad hoc
// through the pool's dialer.
func (s *Service) TestConnection(
	ctx context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	return s.dialer.TestConnection(ctx, in)
}
