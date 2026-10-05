// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsgo

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/convcodecs"
)

// srcDestToEntity maps jetstream Source/Destination onto entity Src/Dest.
var srcDestToEntity = converter.WithFieldMappings(map[string]string{
	"Source":      "Src",
	"Destination": "Dest",
})

// srcDestToJetStream maps entity Src/Dest onto jetstream Source/Destination.
var srcDestToJetStream = converter.WithFieldMappings(map[string]string{
	"Src":  "Source",
	"Dest": "Destination",
})

// streamConvertOpts maps jetstream.StreamInfo → entities.StreamInfo. TimeStamp is set by toStreamInfo.
// RePublish is converted separately: its Src/Dest mapping would also hit SubjectTransformConfig.
var streamConvertOpts = []converter.Option{
	converter.WithIgnoreFields("TimeStamp"),
}

// consumerConvertOpts ignores time fields the converter can't bridge, so
// Created/TimeStamp/OptStartTime are handled explicitly, and renames the
// sequence Last onto LastActive.
var consumerConvertOpts = []converter.Option{
	converter.WithIgnoreFields("OptStartTime", "Created", "TimeStamp"),
	converter.WithFieldMappings(map[string]string{"Last": "LastActive"}),
}

// toStreamInfo converts info and sets Raw, TimeStamp and Republish (see streamConvertOpts).
func toStreamInfo(info *jetstream.StreamInfo) *entities.StreamInfo {
	result := converter.Convert(info, &entities.StreamInfo{}, streamConvertOpts...)
	if info.Config.RePublish != nil {
		result.Config.Republish = converter.Convert(info.Config.RePublish, &entities.StreamRePublish{}, srcDestToEntity)
	}
	rawJSON, _ := json.Marshal(info) //nolint:errcheck // jetstream.StreamInfo marshals deterministically
	result.Raw = string(rawJSON)
	result.TimeStamp = new(time.Now().UTC())
	return result
}

func toJetStreamConsumerConfig(config entities.ConsumerCreateRequest) (*jetstream.ConsumerConfig, error) {
	jsConfig := converter.Convert(config, &jetstream.ConsumerConfig{},
		converter.WithIgnoreFields("OptStartTime"),
	)
	if !config.Ephemeral {
		jsConfig.Durable = jsConfig.Name
	}

	if config.OptStartTime != "" {
		startTime, err := time.Parse(time.RFC3339, config.OptStartTime)
		if err != nil {
			return nil, &errs.NATSValidationError{
				Description: fmt.Sprintf("invalid opt_start_time %q: must be RFC3339", config.OptStartTime),
				Cause:       err,
			}
		}
		jsConfig.OptStartTime = &startTime
	}

	return jsConfig, nil
}

// toConsumerInfo uses a caller-supplied Stream since the SDK doesn't always carry
// the parent stream name.
func toConsumerInfo(info *jetstream.ConsumerInfo, streamName string) *entities.ConsumerInfo {
	result := converter.Convert(info, &entities.ConsumerInfo{}, consumerConvertOpts...)
	rawJSON, _ := json.Marshal(info) //nolint:errcheck // jetstream.ConsumerInfo marshals deterministically
	result.Raw = string(rawJSON)
	result.Stream = streamName
	result.Created = new(info.Created)
	result.TimeStamp = new(info.TimeStamp)
	if info.Config.OptStartTime != nil && result.Config != nil {
		result.Config.OptStartTime = info.Config.OptStartTime.Format(time.RFC3339)
	}
	if info.Paused && info.Config.PauseUntil != nil {
		result.PauseUntil = new(*info.Config.PauseUntil)
	}
	return result
}

// rawMessageOpts maps jetstream.RawStreamMsg → entities.Message. IgnoreZeroValues
// preserves nil-header→nil-map; DataSize/ContentType/DataRawHex set explicitly.
var rawMessageOpts = []converter.Option{
	converter.WithCodecs(
		convcodecs.BytesBase64,
		convcodecs.StringSliceJoin,
	),
	converter.WithFieldMappings(map[string]string{
		"Time":   "Timestamp",
		"Data":   "DataBase64",
		"Header": "Headers",
	}),
	converter.WithIgnoreZeroValues(),
}

// toMessage derives DataSize and ContentType from msg.Data.
func toMessage(msg *jetstream.RawStreamMsg) *entities.Message {
	result := converter.Convert(msg, &entities.Message{}, rawMessageOpts...)
	result.DataSize = len(msg.Data)
	result.ContentType = entities.DetectContentType(msg.Data)
	return result
}

func toMessageWithHex(msg *jetstream.RawStreamMsg) *entities.Message {
	message := toMessage(msg)
	message.DataRawHex = hex.EncodeToString(msg.Data)
	return message
}
