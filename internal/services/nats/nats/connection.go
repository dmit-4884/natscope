// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package nats

import (
	"context"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/natsclient"
)

// GetConnectionURL returns the NATS server URL for the given connection ID.
func (s *Service) GetConnectionURL(ctx context.Context, connectionID string) (string, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return "", err
	}
	return c.URL(), nil
}

// EnsureWritable returns [errs.ErrConnectionReadOnly] for a read-only connection, so a write is refused before
// anything is prepared for it.
func (s *Service) EnsureWritable(ctx context.Context, connectionID string) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	if natsclient.IsReadOnly(c) {
		return errs.ErrConnectionReadOnly
	}
	return nil
}

// GetConnectionHealth returns health status for a specific connection.
func (s *Service) GetConnectionHealth(ctx context.Context, connectionID string) (*entities.ConnectionHealth, error) {
	if c, ok := s.pool.Pooled(connectionID); ok && c.IsReconnecting() {
		return c.Health(ctx)
	}

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
	onDenied func(error),
) (entities.Subscription, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.Subscribe(ctx, subject, handler, onDenied)
}

// MicroInfo collects every NATS Micro service instance's answer to $SRV.INFO.
func (s *Service) MicroInfo(ctx context.Context, connectionID string) ([]entities.MicroReport, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.MicroInfo(ctx)
}

// MicroStats collects every NATS Micro service instance's answer to $SRV.STATS.
func (s *Service) MicroStats(ctx context.Context, connectionID string) ([]entities.MicroReport, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.MicroStats(ctx)
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

// LinkDown reports whether the pooled connection for connectionID has lost its server.
func (s *Service) LinkDown(connectionID string) bool {
	c, ok := s.pool.Pooled(connectionID)
	return ok && !c.IsConnected()
}

// OnDisconnect registers fn to run whenever the pool drops a connection.
func (s *Service) OnDisconnect(fn func(connectionID string)) {
	s.pool.OnDisconnect(fn)
}

// TestConnection tests a NATS connection without saving it, dialing ad hoc
// through the pool's dialer.
func (s *Service) TestConnection(
	ctx context.Context,
	in *entities.TestConnectionRequest,
) (*entities.TestConnectionResult, error) {
	return s.dialer.TestConnection(ctx, in)
}
