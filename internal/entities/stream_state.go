// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// StreamState is the current message/byte counters and sequence bounds for a stream.
type StreamState struct {
	// Msgs is the number of messages.
	Msgs uint64

	// Bytes is the total size in bytes.
	Bytes uint64

	// FirstSeq is the first sequence number.
	FirstSeq uint64

	// LastSeq is the last sequence number.
	LastSeq uint64

	// FirstTime is the timestamp of the first message.
	FirstTime time.Time

	// LastTime is the timestamp of the last message.
	LastTime time.Time

	// Consumers is the number of consumers on this stream.
	Consumers int

	// NumDeleted is the number of deleted messages.
	NumDeleted int

	// NumSubjects is the number of unique subjects.
	NumSubjects uint64
}
