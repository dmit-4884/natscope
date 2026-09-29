// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

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

	if err := validateConsumerRequestLengths(streamName, config.Name, config.FilterSubject, config.FilterSubjects); err != nil {
		return nil, wrapErr(err)
	}
	// Check the trimmed name; a blank durable name becomes ephemeral.
	if !config.Ephemeral && config.Name == "" {
		return nil, wrapErr(&errs.NATSValidationError{Description: "consumer name is required"})
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	jsConfig, err := toJetStreamConsumerConfig(config)
	if err != nil {
		return nil, wrapErr(err)
	}
	if err = c.requireFeatures(consumerConfigFeatures(*jsConfig)...); err != nil {
		return nil, err
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

	if config.FilterSubject != nil && len(config.FilterSubjects) > 0 {
		return nil, wrapErr(&errs.NATSValidationError{Description: "filter_subject and filter_subjects are mutually exclusive"})
	}
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return nil, wrapErr(err)
	}
	// The consumer name goes into a hand-built subject, so reject separator and wildcard characters.
	if err := validateConsumerNameChars(consumerName); err != nil {
		return nil, wrapErr(err)
	}
	if config.FilterSubject != nil {
		if err := validateNATSSubjectLength("filter subject", *config.FilterSubject); err != nil {
			return nil, wrapErr(err)
		}
	}
	for _, s := range config.FilterSubjects {
		if err := validateNATSSubjectLength("filter subject", s); err != nil {
			return nil, wrapErr(err)
		}
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	subject := fmt.Sprintf("$JS.API.CONSUMER.INFO.%s.%s", streamName, consumerName)
	msg, err := c.request(ctx, subject, nil)
	if err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "get consumer info"))
	}

	var infoResp struct {
		Config *jetstream.ConsumerConfig `json:"config"`
		Error  *jetstream.APIError       `json:"error,omitempty"`
	}
	if err = json.Unmarshal(msg.Data, &infoResp); err != nil {
		return nil, wrapErr(errors.WrapOperation(err, "unmarshal consumer info"))
	}
	if infoResp.Error != nil {
		// The hand-built subject bypasses the SDK's not-found translation.
		if infoResp.Error.ErrorCode == jetstream.JSErrCodeConsumerNotFound {
			return nil, wrapErr(jetstream.ErrConsumerNotFound)
		}
		return nil, wrapErr(&errs.NATSAPIError{
			Code:        infoResp.Error.Code,
			ErrorCode:   uint16(infoResp.Error.ErrorCode),
			Description: infoResp.Error.Description,
		})
	}
	if infoResp.Config == nil {
		return nil, wrapErr(errs.ErrConsumerNotFound)
	}

	updatedConfig := applyConsumerUpdate(*infoResp.Config, config)
	if err = c.requireFeatures(consumerConfigFeatures(updatedConfig)...); err != nil {
		return nil, err
	}

	consumer, err := stream.UpdateConsumer(ctx, updatedConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toConsumerInfo(consumer.CachedInfo(), streamName), nil
}

// DeleteConsumer deletes a consumer from a stream.
func (c *Client) DeleteConsumer(ctx context.Context, streamName string, consumerName string) error {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return wrapErr(err)
	}

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
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return nil, wrapErr(err)
	}

	pauseUntilTime, err := time.Parse(time.RFC3339, pauseUntil)
	if err != nil {
		// NATSValidationError keeps the field-specific message that ErrInvalidRequest would replace.
		return nil, wrapErr(&errs.NATSValidationError{
			Description: fmt.Sprintf("invalid pause_until %q: must be RFC3339", pauseUntil),
			Cause:       err,
		})
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	resp, err := stream.PauseConsumer(ctx, consumerName, pauseUntilTime)
	if err != nil {
		return nil, wrapErr(err)
	}

	return &entities.ConsumerPauseResponse{
		Paused:         resp.Paused,
		PauseUntil:     &resp.PauseUntil,
		PauseRemaining: resp.PauseRemaining,
	}, nil
}

// ResumeConsumer resumes a paused consumer and returns the server's post-resume state.
func (c *Client) ResumeConsumer(ctx context.Context, streamName string, consumerName string) (*entities.ConsumerPauseResponse, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}

	resp, err := stream.ResumeConsumer(ctx, consumerName)
	if err != nil {
		return nil, wrapErr(err)
	}

	return &entities.ConsumerPauseResponse{
		Paused:         resp.Paused,
		PauseUntil:     &resp.PauseUntil,
		PauseRemaining: resp.PauseRemaining,
	}, nil
}

// ResetConsumer resets a consumer's delivery state; a nil sequence keeps the
// ack floor, a sequence makes the next delivery start at it.
func (c *Client) ResetConsumer(
	ctx context.Context,
	streamName, consumerName string,
	sequence *uint64,
) (*entities.ConsumerResetResponse, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return nil, wrapErr(err)
	}
	if err := c.requireFeatures(featConsumerReset); err != nil {
		return nil, err
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}
	if _, err = stream.Consumer(ctx, consumerName); err != nil {
		return nil, wrapErr(err)
	}

	var resp *jetstream.ConsumerResetResponse
	if sequence != nil {
		resp, err = stream.ResetConsumerToSequence(ctx, consumerName, *sequence)
	} else {
		resp, err = stream.ResetConsumer(ctx, consumerName)
	}
	if err != nil {
		return nil, wrapErr(err)
	}

	out := &entities.ConsumerResetResponse{ResetSeq: resp.ResetSeq}
	if resp.ConsumerInfo != nil {
		out.Consumer = toConsumerInfo(resp.ConsumerInfo, streamName)
	}
	return out, nil
}

// applyConsumerUpdate merges the set fields of update onto the server's
// current config.
func applyConsumerUpdate(current jetstream.ConsumerConfig, update entities.ConsumerUpdateRequest) jetstream.ConsumerConfig {
	converter.Convert(update, &current,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("FilterSubject", "FilterSubjects"),
	)
	if update.FilterSubject != nil {
		current.FilterSubject = *update.FilterSubject
		current.FilterSubjects = nil
	}
	if len(update.FilterSubjects) > 0 {
		current.FilterSubjects = update.FilterSubjects
		current.FilterSubject = ""
	}
	if update.PriorityPolicy != nil && *update.PriorityPolicy == entities.PriorityNone {
		current.PriorityGroups = nil
		current.PinnedTTL = 0
	}
	return current
}

// UnpinConsumer releases the pinned client of a consumer priority group.
func (c *Client) UnpinConsumer(ctx context.Context, streamName, consumerName, group string) error {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return wrapErr(err)
	}
	if err := validateNATSNameLength("consumer name", consumerName); err != nil {
		return wrapErr(err)
	}
	if err := c.requireFeatures(featPriorityGroups); err != nil {
		return err
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return wrapErr(err)
	}
	if err := stream.UnpinConsumer(ctx, consumerName, group); err != nil {
		return wrapErr(err)
	}
	return nil
}
