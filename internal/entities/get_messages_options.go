// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// GetMessagesOptions holds options for fetching messages.
type GetMessagesOptions struct {
	// StartSeq is the sequence to start from (0 = latest).
	StartSeq uint64

	// StartTime requests the first message at/after this time; resolved to a start
	// sequence. Mutually exclusive with non-zero StartSeq.
	StartTime *time.Time

	// Limit caps returned messages.
	Limit int

	// SubjectFilter filters by subject pattern (wildcards supported).
	SubjectFilter string

	// Direction is the fetch direction (forward, backward).
	Direction string

	// ContentFilter is a case-insensitive substring filter on payload content.
	ContentFilter string

	// FetchMethod is "direct" (parallel GetMsg) or "consumer" (ordered consumer).
	FetchMethod string

	// MaxPayloadBytes caps each preview payload; 0 = unlimited. On cap,
	// Message.Truncated=true and DataSize keeps the original count.
	MaxPayloadBytes int32
}
