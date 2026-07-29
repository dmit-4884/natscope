// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// LiveSubscribeRequest is the input for a live subscription session.
type LiveSubscribeRequest struct {
	ConnectionId  string
	Subscriptions []*LiveSubscriptionTarget
	// MaxPayloadBytes caps each emitted payload; nil → user-settings default
	// (Messages.MaxPayloadBytesInList), 0 → unlimited.
	MaxPayloadBytes *int32
}

// LiveSubscriptionTarget is one subject (with optional stream binding) to
// live-deliver.
type LiveSubscriptionTarget struct {
	Subject    string
	StreamName *string
}

// LiveEvent is a discriminated union of live-service frames; exactly one field
// is non-nil.
type LiveEvent struct {
	Batch       *LiveBatch
	Stats       *LiveStats
	Error       *LiveError
	ProtoReload *LiveProtoReload
}

// LiveBatch carries a batch of messages observed since the previous batch.
type LiveBatch struct {
	Messages []*LiveMessage
}

// LiveMessage is a NatsMessage enriched with optional decoded form; DecodeError
// is set when the decoder ran but failed.
type LiveMessage struct {
	NatsMessage NatsMessage
	Decoded     *string
	DecodedType *string
	DecodeError *string
	// Truncated is set when the server capped the payload for preview; UI loads
	// the full payload via MessagesService.Get(sequence).
	Truncated bool
	// OriginalSize is the pre-truncation byte count, always set so the wire
	// carries it even when Data was shortened; 0 = not measured.
	OriginalSize int
}

// LiveStats is a periodic stats snapshot for the current subscription.
type LiveStats struct {
	TotalMessages     int64
	MessagesPerSecond int64
	SubjectCounts     map[string]int64
	MessagesDropped   int64
}

// LiveError is a non-fatal in-stream notification; fatal errors are returned
// from Service.Subscribe directly.
type LiveError struct {
	Code    string
	Message string
}

// LiveProtoReload signals server-side descriptor reload; the service resets its
// decoders before emitting this.
type LiveProtoReload struct {
	MessagesCount int32
}
