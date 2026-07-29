// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"encoding/json"
	"time"
)

// Message is a decoded NATS JetStream message with its metadata.
type Message struct {
	// Sequence is the stream sequence number.
	Sequence uint64

	// Subject is the message subject.
	Subject string

	// Timestamp is when the message was published.
	Timestamp time.Time

	// DataBase64 is the message data encoded as base64.
	DataBase64 string

	// DataRawHex is the raw hex representation (for single message view).
	DataRawHex string

	// DataSize is the size of the message data in bytes.
	DataSize int

	// ContentType is the detected content type (json, text, binary).
	ContentType ContentType

	// Headers are the message headers.
	Headers map[string]string

	// Decoded protobuf fields (populated when user has mappings configured).
	Decoded     json.RawMessage
	DecodedType string
	DecodeError string

	// Truncated means DataBase64/Decoded were capped for preview; DataSize keeps
	// the original size, full data via Get(sequence).
	Truncated bool
}
