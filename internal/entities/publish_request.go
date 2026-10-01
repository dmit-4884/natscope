// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// PublishRequest is the publish service input; MessageType/SourceID/SchemaFingerprint
// drive JSON→proto encoding, else Data is sent verbatim.
type PublishRequest struct {
	// ConnectionID identifies which saved NATS connection to publish through.
	ConnectionID string `normalize:"trim"`
	// Subject is the literal subject (after wildcard resolution).
	Subject string `normalize:"trim"`
	// Data is the payload: JSON string when MessageType is set, raw text/binary
	// otherwise.
	Data string
	// Headers carries optional NATS message headers.
	Headers map[string]string
	// MessageType, when set, triggers proto encoding of Data via
	// SourceID/SchemaFingerprint.
	MessageType *string `normalize:"trim,nil_on_empty"`
	// SourceID is the proto source to encode against; required when MessageType is
	// set.
	SourceID *string `normalize:"trim,nil_on_empty"`
	// SchemaFingerprint picks a stored schema; empty uses the active one.
	SchemaFingerprint *string `normalize:"trim,nil_on_empty"`
	// SubjectPattern is the UI subject template, for history bookkeeping; not used
	// during publish.
	SubjectPattern *string `normalize:"trim,nil_on_empty"`
}

// PublishResult is the outcome of a publish; soft-failure: all errors surface
// via Error, Stream/Sequence valid only when Error is nil.
type PublishResult struct {
	Stream    string
	Sequence  uint64
	Duplicate bool
	Error     *string
	// CounterValue is the new counter total after a Nats-Incr publish.
	CounterValue *string
}
