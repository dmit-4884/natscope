// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"
	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// ListStreams returns all JetStream streams via one paged STREAM.LIST call.
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

// ListStreamNames returns the names of all streams, sorted; like ListStreams it
// skips internal "$"-prefixed streams.
func (c *Client) ListStreamNames(ctx context.Context) ([]string, error) {
	names := []string{}
	nameLister := c.jetStream.StreamNames(ctx)
	for name := range nameLister.Name() {
		if strings.HasPrefix(name, "$") {
			continue
		}
		names = append(names, name)
	}
	if err := nameLister.Err(); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "list stream names"))
	}

	sort.Strings(names)
	return names, nil
}

// streamListPage is one page of the STREAM.LIST response; streams stay raw so each keeps the server's JSON.
type streamListPage struct {
	Total   int                 `json:"total"`
	Streams []json.RawMessage   `json:"streams"`
	Error   *jetstream.APIError `json:"error,omitempty"`
}

// streamTopologyInfo is a stream info whose mirror and source states keep the external and error
// fields that jetstream.StreamSourceInfo drops.
type streamTopologyInfo struct {
	jetstream.StreamInfo
	Mirror  *streamLinkInfo   `json:"mirror,omitempty"`
	Sources []*streamLinkInfo `json:"sources,omitempty"`
}

type streamLinkInfo struct {
	jetstream.StreamSourceInfo
	External *jetstream.ExternalStream `json:"external,omitempty"`
	Error    *jetstream.APIError       `json:"error,omitempty"`
}

// ListStreamTopology returns every stream with the live state of its mirror and source links (lag,
// last activity, errors). It pages STREAM.LIST itself: the SDK lister drops the link errors.
func (c *Client) ListStreamTopology(ctx context.Context) ([]entities.StreamInfo, error) {
	streams := []entities.StreamInfo{}
	for offset := 0; ; {
		page, err := c.streamListPage(ctx, offset)
		if err != nil {
			return nil, err
		}
		for _, raw := range page.Streams {
			var s streamTopologyInfo
			if err := json.Unmarshal(raw, &s); err != nil {
				return nil, wrapErr(coreerrs.WrapOperation(err, "unmarshal stream info"))
			}
			if strings.HasPrefix(s.Config.Name, "$") {
				continue
			}
			info := toStreamInfo(&s.StreamInfo)
			info.Raw = string(raw)
			info.Mirror = toSourceInfo(s.Mirror)
			info.Sources = slices.To(s.Sources, toSourceInfo)
			streams = append(streams, *info)
		}
		offset += len(page.Streams)
		if len(page.Streams) == 0 || offset >= page.Total {
			return streams, nil
		}
	}
}

func (c *Client) streamListPage(ctx context.Context, offset int) (*streamListPage, error) {
	reqData, err := json.Marshal(struct {
		Offset int `json:"offset"`
	}{Offset: offset})
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "marshal stream list request"))
	}

	msg, err := c.request(ctx, c.apiSubject("STREAM.LIST"), reqData)
	if err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "list streams"))
	}

	var page streamListPage
	if err := json.Unmarshal(msg.Data, &page); err != nil {
		return nil, wrapErr(coreerrs.WrapOperation(err, "unmarshal stream list"))
	}
	if page.Error != nil {
		return nil, wrapErr(&errs.NATSAPIError{
			Code:        page.Error.Code,
			ErrorCode:   uint16(page.Error.ErrorCode),
			Description: page.Error.Description,
		})
	}
	return &page, nil
}

func toSourceInfo(link *streamLinkInfo) *entities.StreamSourceInfo {
	if link == nil {
		return nil
	}
	info := converter.Convert(&link.StreamSourceInfo, &entities.StreamSourceInfo{})
	if link.External != nil {
		info.External = converter.Convert(link.External, &entities.ExternalStreamRef{})
	}
	if link.Error != nil {
		info.Error = link.Error.Description
	}
	return info
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

	return toStreamDetail(stream.CachedInfo()), nil
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
		if info == nil || c.isOwnConsumer(info.Name) {
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

	// WithSubjectFilter populates State.Subjects.
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
	rawName := config.Name
	_ = normalizer.Normalize(&config) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := validateNATSNameLength("stream name", config.Name); err != nil {
		return nil, wrapErr(err)
	}
	// Lookups don't trim, so reject a padded name instead of trimming it.
	if rawName != config.Name {
		return nil, wrapErr(&errs.NATSValidationError{
			Description: fmt.Sprintf("stream name %q must not have leading/trailing whitespace", rawName),
		})
	}
	// "$"-prefixed names are reserved for system streams and hidden from lists.
	if strings.HasPrefix(config.Name, "$") {
		return nil, wrapErr(&errs.NATSValidationError{
			Description: fmt.Sprintf("stream name %q must not start with '$' (reserved for system streams)", config.Name),
		})
	}

	jsConfig := converter.Convert(config, &jetstream.StreamConfig{}, srcDestToJetStream)
	if err := c.requireFeatures(ctx, streamConfigFeatures(*jsConfig)...); err != nil {
		return nil, err
	}

	stream, err := c.jetStream.CreateStream(ctx, *jsConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toStreamDetail(stream.CachedInfo()), nil
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
	if err = c.requireFeatures(ctx, streamConfigFeatures(updatedConfig)...); err != nil {
		return nil, err
	}

	stream, err = c.jetStream.UpdateStream(ctx, updatedConfig)
	if err != nil {
		return nil, wrapErr(err)
	}

	return toStreamDetail(stream.CachedInfo()), nil
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
	if req.Sequence > 0 && req.Keep > 0 {
		return 0, &errs.NATSValidationError{Description: "a purge takes either a sequence or a number of messages to keep, not both"}
	}

	stream, err := c.jetStream.Stream(ctx, name)
	if err != nil {
		return 0, wrapErr(coreerrs.WrapOperation(err, "get stream"))
	}

	// Fail fast on deny_purge/sealed, as DeleteMessage does for deny_delete.
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

	msg, err := c.request(ctx, c.apiSubject("STREAM.PURGE."+name), reqData)
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

	return toStreamDetail(stream.CachedInfo()), nil
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

	// Replace, not append: the UI sends back the current sources on every save; nil leaves them unchanged.
	if len(update.Sources) > 0 {
		current.Sources = slices.To(update.Sources, func(src *entities.StreamSource) *jetstream.StreamSource {
			return converter.Convert(src, &jetstream.StreamSource{})
		})
	}

	if update.Republish != nil {
		// An empty Republish clears it; converted as-is, NATS would treat it as a ">"→">" passthrough.
		if update.Republish.Src == "" && update.Republish.Dest == "" {
			current.RePublish = nil
		} else {
			current.RePublish = converter.Convert(update.Republish, &jetstream.RePublish{}, srcDestToJetStream)
		}
	}

	return current
}
