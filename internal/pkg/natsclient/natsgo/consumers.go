// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/altessa-s/go-atlas/core/errors"
	"github.com/altessa-s/go-atlas/core/runtime/concurrency"
	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	slogx "github.com/altessa-s/go-atlas/observability/slog"
)

// GetAllConsumers returns all consumers across all streams using parallel
// fetching.
func (c *Client) GetAllConsumers(ctx context.Context) ([]entities.ConsumerStats, error) {
	streamLister := c.jetStream.ListStreams(ctx)
	var streamNames []string //nolint:prealloc
	for streamInfo := range streamLister.Info() {
		streamNames = append(streamNames, streamInfo.Config.Name)
	}
	if err := streamLister.Err(); err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "list streams"))
	}

	// Best-effort: a stream whose consumers fail to load is logged and skipped
	// (fn returns nil), never aborting the batch.
	perStream, collectErr := concurrency.ProcessCollect(ctx, streamNames,
		func(ctx context.Context, streamName string) ([]entities.ConsumerStats, error) {
			defer panics.Handle(ctx)
			consumers, err := c.fetchStreamConsumersStats(ctx, streamName)
			if err != nil {
				c.logger.Warn("error fetching consumers", slogx.Error(err))
				return nil, nil
			}
			return consumers, nil
		},
	)
	if collectErr != nil {
		return nil, wrapErr(collectErr)
	}

	var allConsumers []entities.ConsumerStats //nolint:prealloc
	for _, consumers := range perStream {
		allConsumers = append(allConsumers, consumers...)
	}

	return allConsumers, nil
}

func (c *Client) fetchStreamConsumersStats(ctx context.Context, streamName string) ([]entities.ConsumerStats, error) {
	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(errors.Wrapf(err, "failed to get stream %s", streamName))
	}

	var consumers []entities.ConsumerStats //nolint:prealloc
	consumerLister := stream.ListConsumers(ctx)

	for info := range consumerLister.Info() {
		consumer := converter.Convert(info, &entities.ConsumerStats{})
		consumer.Stream = streamName

		// OptStartTime (*time.Time in SDK config) can't be bridged by the converter,
		// so it's set explicitly below.
		converter.Convert(&info.Config, consumer,
			converter.WithIgnoreFields("Name", "OptStartTime"),
		)
		if info.Config.OptStartTime != nil {
			consumer.OptStartTime = *info.Config.OptStartTime
		}

		consumers = append(consumers, *consumer)
	}

	if err := consumerLister.Err(); err != nil {
		return consumers, wrapErr(errors.Wrapf(err, "failed to list consumers for stream %s", streamName))
	}

	return consumers, nil
}

// CreateConsumer creates a new consumer on a stream.
func (c *Client) CreateConsumer(
	ctx context.Context,
	streamName string,
	config entities.ConsumerCreateRequest,
) (*entities.ConsumerInfo, error) {
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	jsConfig, err := toJetStreamConsumerConfig(config)
	if err != nil {
		return nil, wrapErr(err)
	}

	consumer, err := stream.CreateConsumer(ctx, *jsConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toConsumerInfo(consumer.CachedInfo(), streamName), nil
}

// UpdateConsumer updates an existing consumer configuration.
func (c *Client) UpdateConsumer(
	ctx context.Context,
	streamName, consumerName string,
	config entities.ConsumerUpdateRequest,
) (*entities.ConsumerInfo, error) {
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	consumer, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		return nil, wrapErr(err)
	}

	currentInfo := consumer.CachedInfo()
	updatedConfig := currentInfo.Config
	converter.Convert(config, &updatedConfig, converter.WithIgnoreNilValues())

	consumer, err = stream.UpdateConsumer(ctx, updatedConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toConsumerInfo(consumer.CachedInfo(), streamName), nil
}

// DeleteConsumer deletes a consumer from a stream.
func (c *Client) DeleteConsumer(ctx context.Context, streamName string, consumerName string) error {
	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return wrapErr(err)
	}

	if err := stream.DeleteConsumer(ctx, consumerName); err != nil {
		return wrapErr(err)
	}

	return nil
}

// PauseConsumer pauses a consumer until the supplied RFC3339 timestamp.
func (c *Client) PauseConsumer(
	ctx context.Context,
	streamName, consumerName string,
	pauseUntil string,
) (*entities.ConsumerPauseResponse, error) {
	pauseUntilTime, err := time.Parse(time.RFC3339, pauseUntil)
	if err != nil {
		return nil, wrapErr(fmt.Errorf("%w: invalid pause_until (must be RFC3339): %v", errs.ErrInvalidRequest, err))
	}

	pauseReq := struct {
		PauseUntil time.Time `json:"pause_until"`
	}{PauseUntil: pauseUntilTime}

	reqData, err := json.Marshal(pauseReq)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "marshal pause request"))
	}

	subject := fmt.Sprintf("$JS.API.CONSUMER.PAUSE.%s.%s", streamName, consumerName)
	msg, err := c.request(ctx, subject, reqData)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "pause consumer"))
	}

	var resp struct {
		Paused         bool      `json:"paused"`
		PauseUntil     time.Time `json:"pause_until"`
		PauseRemaining int64     `json:"pause_remaining"`
		Error          *struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "unmarshal pause response"))
	}
	if resp.Error != nil {
		return nil, wrapErr(&errs.NATSAPIError{
			Code:        resp.Error.Code,
			Description: resp.Error.Description,
		})
	}

	return &entities.ConsumerPauseResponse{
		Paused:         resp.Paused,
		PauseUntil:     &resp.PauseUntil,
		PauseRemaining: time.Duration(resp.PauseRemaining),
	}, nil
}

// ResumeConsumer resumes a paused consumer immediately.
func (c *Client) ResumeConsumer(ctx context.Context, streamName string, consumerName string) error {
	subject := fmt.Sprintf("$JS.API.CONSUMER.PAUSE.%s.%s", streamName, consumerName)
	msg, err := c.request(ctx, subject, []byte("{}"))
	if err != nil {
		return wrapErr(errors.WrapOperation(err, "resume consumer"))
	}

	var resp struct {
		Error *struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return wrapErr(errors.WrapOperation(err, "unmarshal resume response"))
	}
	if resp.Error != nil {
		return wrapErr(&errs.NATSAPIError{
			Code:        resp.Error.Code,
			Description: resp.Error.Description,
		})
	}

	return nil
}
