// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// ListStreams returns all JetStream streams via a single paged STREAM.LIST
// call (O(1) JetStream API requests), matching GetAllStreamsStats instead of
// the previous STREAM.NAMES + one STREAM.INFO per stream (QA-116).
func (c *Client) ListStreams(ctx context.Context) ([]entities.StreamInfo, error) {
	streams := []entities.StreamInfo{}
	streamLister := c.jetStream.ListStreams(ctx)
	for info := range streamLister.Info() {
		if info == nil || strings.HasPrefix(info.Config.Name, "$") {
			continue
		}
		streams = append(streams, *toStreamInfo(info))
	}
	if err := streamLister.Err(); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "list streams"))
	}

	return streams, nil
}

// GetStreamInfo returns detailed information about a specific stream.
func (c *Client) GetStreamInfo(ctx context.Context, streamName string) (*entities.StreamInfo, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	return toStreamInfo(stream.CachedInfo()), nil
}

// GetStreamConsumers returns detailed consumer information for a stream.
func (c *Client) GetStreamConsumers(ctx context.Context, streamName string) ([]entities.ConsumerInfo, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	consumers := []entities.ConsumerInfo{}
	consumerLister := stream.ListConsumers(ctx)
	for info := range consumerLister.Info() {
		if info == nil {
			continue
		}
		consumers = append(consumers, *toConsumerInfo(info, streamName))
	}
	if err := consumerLister.Err(); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "list consumers"))
	}

	return consumers, nil
}

// GetAllStreamsStats returns statistics for all streams in the connection.
func (c *Client) GetAllStreamsStats(ctx context.Context) ([]entities.StreamStats, error) {
	streamLister := c.jetStream.ListStreams(ctx)
	var stats []entities.StreamStats //nolint:prealloc

	for info := range streamLister.Info() {
		if strings.HasPrefix(info.Config.Name, "$") {
			continue
		}
		stat := entities.StreamStats{
			Name:          info.Config.Name,
			Messages:      info.State.Msgs,
			Bytes:         info.State.Bytes,
			FirstSeq:      info.State.FirstSeq,
			LastSeq:       info.State.LastSeq,
			ConsumerCount: info.State.Consumers,
			Created:       info.Created,
		}

		if info.Cluster != nil {
			stat.Cluster = &entities.ClusterStats{
				Name:   info.Cluster.Name,
				Leader: info.Cluster.Leader,
			}
		}

		stats = append(stats, stat)
	}

	if err := streamLister.Err(); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "list streams"))
	}

	return stats, nil
}

// GetStreamStats returns detailed statistics for a specific stream.
func (c *Client) GetStreamStats(ctx context.Context, streamName string) (*entities.StreamStats, error) {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return nil, wrapErr(coreerrs.Wrap(err, "stream not found"))
	}

	// WithSubjectFilter populates State.Subjects (per-subject message counts);
	// without it the field is always empty (QA-122).
	info, err := stream.Info(ctx, jetstream.WithSubjectFilter(">"))
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "get stream info"))
	}

	stat := &entities.StreamStats{
		Name:          info.Config.Name,
		Messages:      info.State.Msgs,
		Bytes:         info.State.Bytes,
		FirstSeq:      info.State.FirstSeq,
		LastSeq:       info.State.LastSeq,
		ConsumerCount: info.State.Consumers,
		Created:       info.Created,
	}

	if len(info.State.Subjects) > 0 {
		stat.Subjects = info.State.Subjects
	}

	if info.Cluster != nil {
		stat.Cluster = &entities.ClusterStats{
			Name:     info.Cluster.Name,
			Leader:   info.Cluster.Leader,
			Replicas: slices.To(info.Cluster.Replicas, func(r *jetstream.PeerInfo) string { return r.Name }),
		}
	}

	return stat, nil
}

// CreateStream creates a new JetStream stream with the given configuration.
func (c *Client) CreateStream(ctx context.Context, config entities.StreamCreateRequest) (*entities.StreamInfo, error) {
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := validateNATSNameLength("stream name", config.Name); err != nil {
		return nil, wrapErr(err)
	}
	// ListStreams/GetAllStreamsStats hide "$"-prefixed names (reserved for
	// system streams); creating one there would make it invisible in the UI
	// (QA-052).
	if strings.HasPrefix(config.Name, "$") {
		return nil, wrapErr(&errs.NATSValidationError{
			Description: fmt.Sprintf("stream name %q must not start with '$' (reserved for system streams)", config.Name),
		})
	}

	jsConfig := converter.Convert(config, &jetstream.StreamConfig{}, srcDestToJetStream)

	stream, err := c.jetStream.CreateStream(ctx, *jsConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toStreamInfo(stream.CachedInfo()), nil
}

// UpdateStream updates an existing JetStream stream configuration.
func (c *Client) UpdateStream(
	ctx context.Context,
	name string,
	config entities.StreamUpdateRequest,
) (*entities.StreamInfo, error) {
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := validateNATSNameLength("stream name", name); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, name)
	if err != nil {
		return nil, wrapErr(err)
	}

	currentInfo := stream.CachedInfo()
	updatedConfig := c.mergeStreamUpdate(currentInfo.Config, config)

	stream, err = c.jetStream.UpdateStream(ctx, updatedConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toStreamInfo(stream.CachedInfo()), nil
}

// DeleteStream deletes a JetStream stream and all its data.
func (c *Client) DeleteStream(ctx context.Context, name string) error {
	if err := validateNATSNameLength("stream name", name); err != nil {
		return wrapErr(err)
	}

	if err := c.jetStream.DeleteStream(ctx, name); err != nil {
		return wrapErr(err)
	}

	return nil
}

// PurgeStream removes messages from a stream based on the purge request
// options.
func (c *Client) PurgeStream(ctx context.Context, name string, req entities.StreamPurgeRequest) (uint64, error) {
	if err := validateNATSNameLength("stream name", name); err != nil {
		return 0, wrapErr(err)
	}

	// Resolve+validate the stream via the SDK first, instead of building the
	// purge subject by hand against an unchecked name: it rejects a
	// dotted/space-containing name instantly (QA-050 — the hand-built subject
	// otherwise matches no JetStream API responder and hangs for the full
	// request timeout) and gives the usual NATS_STREAM_NOT_FOUND reason for a
	// missing stream instead of a generic NATS_API_ERROR (QA-115).
	stream, err := c.jetStream.Stream(ctx, name)
	if err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	// Fail fast on deny_purge/sealed with a clear precondition error rather
	// than the server's opaque "stream purge not permitted" (mirrors
	// DeleteMessage's deny_delete check).
	if info := stream.CachedInfo(); info != nil && (info.Config.DenyPurge || info.Config.Sealed) {
		return 0, errs.ErrStreamPurgeDenied
	}

	purgeReq := struct {
		Filter   string `json:"filter,omitempty"`
		Sequence uint64 `json:"seq,omitempty"`
		Keep     uint64 `json:"keep,omitempty"`
	}{
		Filter:   req.Filter,
		Sequence: req.Sequence,
		Keep:     req.Keep,
	}

	reqData, err := json.Marshal(purgeReq)
	if err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "marshal purge request"))
	}

	subject := fmt.Sprintf("$JS.API.STREAM.PURGE.%s", name)
	msg, err := c.request(ctx, subject, reqData)
	if err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "purge stream"))
	}

	var resp struct {
		Success bool                `json:"success"`
		Purged  uint64              `json:"purged"`
		Error   *jetstream.APIError `json:"error,omitempty"`
	}
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "unmarshal purge response"))
	}
	if resp.Error != nil {
		return 0, wrapErr(&errs.NATSAPIError{
			Code:        resp.Error.Code,
			ErrorCode:   uint16(resp.Error.ErrorCode),
			Description: resp.Error.Description,
		})
	}

	return resp.Purged, nil
}

// DeleteMessage deletes one message by sequence; secure=true overwrites data
// first (irreversible erase).
func (c *Client) DeleteMessage(ctx context.Context, streamName string, sequence uint64, secure bool) error {
	if err := validateNATSNameLength("stream name", streamName); err != nil {
		return wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, streamName)
	if err != nil {
		return wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	// Fail fast on deny_delete/sealed with a clear precondition error rather than
	// the server's opaque "delete unsuccessful".
	if info := stream.CachedInfo(); info != nil && (info.Config.DenyDelete || info.Config.Sealed) {
		return errs.ErrMsgDeleteDenied
	}

	if secure {
		err = stream.SecureDeleteMsg(ctx, sequence)
	} else {
		err = stream.DeleteMsg(ctx, sequence)
	}
	if err != nil {
		// SDK collapses a missing sequence into ErrMsgDeleteUnsuccessful; confirm
		// with a point read for a clean NotFound.
		if errors.Is(err, jetstream.ErrMsgDeleteUnsuccessful) {
			if _, getErr := stream.GetMsg(ctx, sequence); errors.Is(getErr, jetstream.ErrMsgNotFound) {
				return errs.ErrMsgNotFound
			}
		}
		return wrapErr(err)
	}

	return nil
}

// SealStream seals a stream, making it read-only, and returns its updated info.
func (c *Client) SealStream(ctx context.Context, name string) (*entities.StreamInfo, error) {
	if err := validateNATSNameLength("stream name", name); err != nil {
		return nil, wrapErr(err)
	}

	stream, err := c.jetStream.Stream(ctx, name)
	if err != nil {
		return nil, wrapErr(err)
	}

	currentInfo := stream.CachedInfo()
	sealedConfig := currentInfo.Config
	sealedConfig.Sealed = true

	stream, err = c.jetStream.UpdateStream(ctx, sealedConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toStreamInfo(stream.CachedInfo()), nil
}

func (c *Client) mergeStreamUpdate(
	current jetstream.StreamConfig,
	update entities.StreamUpdateRequest,
) jetstream.StreamConfig {
	// Republish/Sources need custom handling below; the rest maps directly.
	converter.Convert(update, &current,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("Sources", "Republish"),
		srcDestToJetStream,
	)

	// Replace, not append: the UI round-trips the current sources back on
	// every save (passthrough), and re-sending the existing list used to fail
	// with "duplicate source configuration detected" — the only way to ever
	// change a stream's sources was to hit that error (QA-013). A caller that
	// wants sources left untouched simply omits the field (nil slice); an
	// empty (non-nil) list would need proto presence tracking to distinguish
	// "clear all" from "not sent", which is out of scope here.
	if len(update.Sources) > 0 {
		current.Sources = slices.To(update.Sources, func(src *entities.StreamSource) *jetstream.StreamSource {
			return converter.Convert(src, &jetstream.StreamSource{})
		})
	}

	if update.Republish != nil {
		// The proto doc says "send an empty message to clear" (Republish{}),
		// but Convert turns an empty entity into &RePublish{Source:"",
		// Destination:""}, which NATS normalizes to a ">"→">" passthrough
		// route instead of clearing it — every incoming message gets
		// republished to its own subject, filling the stream with copies of
		// itself (QA-001). Treat "no src/dest" as the documented clear.
		if update.Republish.Src == "" && update.Republish.Dest == "" {
			current.RePublish = nil
		} else {
			current.RePublish = converter.Convert(update.Republish, &jetstream.RePublish{}, srcDestToJetStream)
		}
	}

	return current
}
