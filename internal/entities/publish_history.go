// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// HistoryPayloadPreviewBytes caps PayloadJSON per row to a UI preview; original
// size kept in PayloadSize/PayloadTruncated.
const HistoryPayloadPreviewBytes = 4 * 1024

// EncodingType is how the message was encoded for publishing.
type EncodingType string

const (
	EncodingTypeJSON     EncodingType = "json"
	EncodingTypeProtobuf EncodingType = "protobuf"
)

// PublishHistory is a published message entry in history.
type PublishHistory struct {
	BaseEntity

	// ConnectionID is the connection used (optional, for linking).
	ConnectionID *string

	// ConnectionURL is the NATS server URL.
	ConnectionURL string

	// Stream is the JetStream stream name.
	Stream string

	// Subject is the message subject.
	Subject string

	// SubjectPattern is the pattern used for the subject (optional).
	SubjectPattern *string

	// EncodingType is how the message was encoded.
	EncodingType EncodingType

	// MessageType is the protobuf message type used.
	MessageType string

	// PayloadJSON is the payload, capped to HistoryPayloadPreviewBytes; see
	// PayloadTruncated/PayloadSize for original.
	PayloadJSON string

	// PayloadTruncated is true when PayloadJSON was capped at write time.
	PayloadTruncated bool

	// PayloadSize is the original encoded payload size in bytes before any preview
	// truncation.
	PayloadSize int

	// Sequence is the JetStream sequence number (if published successfully).
	Sequence *uint64

	// Success indicates if the publish was successful.
	Success bool

	// Error is the failure message, nil on success.
	Error *string
}

// PublishHistoryNew creates a new PublishHistory with generated Id and
// timestamps.
func PublishHistoryNew(init ...func(*PublishHistory)) *PublishHistory {
	h := &PublishHistory{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](h)
	}

	return h
}

// PublishHistories is a slice of PublishHistory pointers.
type PublishHistories []*PublishHistory

// PublishHistoryList is the listing filter for publish history entries.
type PublishHistoryList struct {
	ListBase
	ConnectionURL *string
	Stream        *string
}

// PublishHistoryCreate is the create DTO for a publish history entry.
type PublishHistoryCreate struct {
	ConnectionID   *string      `normalize:"trim,nil_on_empty"`
	ConnectionURL  string       `normalize:"trim"`
	Stream         string       `normalize:"trim"`
	Subject        string       `normalize:"trim"`
	SubjectPattern *string      `normalize:"trim,nil_on_empty"`
	EncodingType   EncodingType `normalize:"trim"`
	MessageType    string       `normalize:"trim"`
	PayloadJSON    string
	PayloadSize    int
	Sequence       *uint64
	Success        bool
	Error          *string
}
