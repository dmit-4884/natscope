// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package nats

import (
	"context"
	"iter"
	"time"

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

// GetNextMessage returns the first message at or after startSeq on any of subjects, or nil.
func (s *Service) GetNextMessage(
	ctx context.Context,
	connectionID string,
	streamName string,
	startSeq uint64,
	subjects []string,
) (*entities.Message, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.GetNextMessage(ctx, streamName, startSeq, subjects)
}

// Publish sends a core NATS message and flushes it.
func (s *Service) Publish(
	ctx context.Context,
	connectionID string,
	subject string,
	data []byte,
	headers map[string]string,
) error {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return err
	}
	return c.Publish(ctx, subject, data, headers)
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

// Request sends a core NATS request and returns the first reply.
func (s *Service) Request(
	ctx context.Context,
	connectionID string,
	subject string,
	data []byte,
	headers map[string]string,
) (*entities.Reply, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return c.Request(ctx, subject, data, headers)
}

// ScanMessages reads the stored messages in opts' range oldest first.
func (s *Service) ScanMessages(
	ctx context.Context,
	connectionID, streamName string,
	opts entities.ScanOptions,
) iter.Seq2[*entities.Message, error] {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return func(yield func(*entities.Message, error) bool) { yield(nil, err) }
	}
	return c.ScanMessages(ctx, streamName, opts)
}

// SeqAtTime returns the first sequence stored at or after t, or the last sequence plus one when none is.
func (s *Service) SeqAtTime(ctx context.Context, connectionID, streamName, fetchMethod string, t time.Time) (uint64, error) {
	c, err := s.client(ctx, connectionID)
	if err != nil {
		return 0, err
	}
	return c.SeqAtTime(ctx, streamName, fetchMethod, t)
}
