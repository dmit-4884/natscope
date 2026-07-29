// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// MessageListRequest is the input for the messages service's List method;
// optional fields fall back to user settings then NATS defaults.
type MessageListRequest struct {
	ConnectionID string
	StreamName   string
	StartSeq     *uint64
	// StartTime jumps to the first message at/after this publish time; mutually
	// exclusive with StartSeq (enforced by proto validation).
	StartTime     *time.Time
	Limit         *int64
	SubjectFilter *string
	// Direction is "", "forward", or "backward"; empty = user setting then NATS
	// default.
	Direction string
	// FetchMethod is "", "direct", or "consumer"; empty = user setting then NATS
	// default.
	FetchMethod   string
	ContentFilter *string
	// MaxPayloadBytes caps payload + decoded JSON per message; nil → user-settings
	// default (Messages.MaxPayloadBytesInList), 0 → unlimited.
	MaxPayloadBytes *int32
}

// MessageGetRequest is the input for the messages service's Get method.
type MessageGetRequest struct {
	ConnectionID string
	StreamName   string
	Sequence     uint64
}
