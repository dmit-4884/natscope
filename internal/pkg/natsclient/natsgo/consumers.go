// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/runtime/concurrency"
	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// Name prefixes of the short-lived consumers natscope creates to read streams.
const (
	liveConsumerPrefix    = "natscope-live-"
	browseConsumerPrefix  = "natscope-browse-"
	timeResConsumerPrefix = "natscope-timeres-"
)

// trackOwnConsumer remembers a consumer this client creates to read a stream; the returned func forgets it.
func (c *Client) trackOwnConsumer(name string) func() {
	c.ownConsumers.Store(name, struct{}{})
	return func() { c.ownConsumers.Delete(name) }
}

// trackOwnConsumerSeries remembers the consumers an ordered consumer named prefix creates one after another, named
// prefix_1, prefix_2 and so on; the returned func forgets them.
func (c *Client) trackOwnConsumerSeries(prefix string) func() {
	c.ownConsumerSeries.Store(prefix+"_", struct{}{})
	return func() { c.ownConsumerSeries.Delete(prefix + "_") }
}

// isOwnConsumer reports whether this client created the consumer to read a stream.
func (c *Client) isOwnConsumer(name string) bool {
	if _, ok := c.ownConsumers.Load(name); ok {
		return true
	}
	own := false
	c.ownConsumerSeries.Range(func(key, _ any) bool {
		prefix, ok := key.(string)
		own = ok && strings.HasPrefix(name, prefix)
		return !own
	})
	return own
}

// GetConsumersOverview lists every stream, then the consumers of each in parallel; a stream whose consumers
// cannot be listed is reported in UnreadableStreams instead of failing the call. Raw JSON is left out to keep
// a frequent poll small.
func (c *Client) GetConsumersOverview(ctx context.Context) (*entities.ConsumersOverview, error) {
	streams, err := c.ListStreams(ctx)
	if err != nil {
		return nil, err
	}

	type streamConsumers struct {
		consumers []entities.ConsumerInfo
		err       error
	}
	perStream, err := concurrency.ProcessCollect(ctx, streams,
		func(ctx context.Context, stream entities.StreamInfo) (streamConsumers, error) {
			defer panics.Handle(ctx)
			if stream.State.Consumers == 0 {
				return streamConsumers{}, nil
			}
			consumers, listErr := c.listConsumers(ctx, stream.Config.Name)
			return streamConsumers{consumers: consumers, err: listErr}, nil
		},
	)
	if err != nil {
		return nil, wrapErr(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, wrapErr(err)
	}

	overview := &entities.ConsumersOverview{Consumers: []entities.ConsumerInfo{}, Streams: []entities.StreamInfo{}}
	for i, listed := range perStream {
		stream := streams[i]
		switch {
		case errors.Is(listed.err, errs.ErrStreamNotFound):
			continue
		case listed.err != nil:
			overview.UnreadableStreams = append(overview.UnreadableStreams, unreadableStream(stream.Config.Name, listed.err))
		default:
			for _, consumer := range listed.consumers {
				consumer.Raw = ""
				overview.Consumers = append(overview.Consumers, consumer)
			}
		}
		stream.Raw = ""
		overview.Streams = append(overview.Streams, stream)
	}
	return overview, nil
}

// consumerListPage is one page of the CONSUMER.LIST response.
type consumerListPage struct {
	Total     int                       `json:"total"`
	Consumers []*jetstream.ConsumerInfo `json:"consumers"`
	Error     *jetstream.APIError       `json:"error,omitempty"`
}

// listConsumers pages CONSUMER.LIST of a stream whose info the caller already has, so no STREAM.INFO precedes it.
func (c *Client) listConsumers(ctx context.Context, streamName string) ([]entities.ConsumerInfo, error) {
	consumers := []entities.ConsumerInfo{}
	for offset := 0; ; {
		reqData, err := json.Marshal(struct {
			Offset int `json:"offset"`
		}{Offset: offset})
		if err != nil {
			return nil, wrapErr(coreerrs.WrapOperation(err, "marshal consumer list request"))
		}
		msg, err := c.request(ctx, c.apiSubject("CONSUMER.LIST."+streamName), reqData)
		if err != nil {
			return nil, wrapErr(coreerrs.WrapOperation(err, "list consumers"))
		}
		var page consumerListPage
		if err := json.Unmarshal(msg.Data, &page); err != nil {
			return nil, wrapErr(coreerrs.WrapOperation(err, "unmarshal consumer list"))
		}
		if page.Error != nil {
			return nil, wrapErr(page.Error)
		}
		for _, info := range page.Consumers {
			if info == nil || c.isOwnConsumer(info.Name) {
				continue
			}
			consumers = append(consumers, *toConsumerInfo(info, streamName))
		}
		offset += len(page.Consumers)
		if len(page.Consumers) == 0 || offset >= page.Total {
			return consumers, nil
		}
	}
}

func unreadableStream(name string, err error) entities.UnreadableStream {
	if permErr, ok := errors.AsType[*errs.NATSPermissionError](err); ok {
		return entities.UnreadableStream{
			Stream: name,
			Access: &entities.AccessCheck{Status: entities.AccessDenied, Operation: permErr.Operation, Subject: permErr.Subject},
		}
	}
	return entities.UnreadableStream{Stream: name, Err: err}
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
	if err = c.requireFeatures(ctx, consumerConfigFeatures(*jsConfig)...); err != nil {
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

	msg, err := c.request(ctx, c.apiSubject("CONSUMER.INFO."+streamName+"."+consumerName), nil)
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "get consumer info"))
	}

	var infoResp struct {
		Config *jetstream.ConsumerConfig `json:"config"`
		Error  *jetstream.APIError       `json:"error,omitempty"`
	}
	if err = json.Unmarshal(msg.Data, &infoResp); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "unmarshal consumer info"))
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
	if err = c.requireFeatures(ctx, consumerConfigFeatures(updatedConfig)...); err != nil {
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
	if !pauseUntilTime.After(time.Now()) {
		return nil, &errs.NATSValidationError{
			Description: fmt.Sprintf("pause_until %s is not in the future; resume the consumer instead", pauseUntil),
		}
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
	if err := c.requireFeatures(ctx, featConsumerReset); err != nil {
		return nil, err
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(err)
	}
	if _, err = stream.Consumer(ctx, consumerName); err != nil {
		return nil, wrapErr(err)
	}
	if info := stream.CachedInfo(); sequence != nil && info != nil && *sequence > info.State.LastSeq+1 {
		return nil, &errs.NATSValidationError{
			Description: fmt.Sprintf("sequence %d is beyond the stream's last sequence %d", *sequence, info.State.LastSeq),
		}
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
	if err := c.requireFeatures(ctx, featPriorityGroups); err != nil {
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
