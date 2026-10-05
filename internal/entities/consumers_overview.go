// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ConsumersOverview is every consumer the user can read across all streams, with the streams themselves.
type ConsumersOverview struct {
	Consumers         []ConsumerInfo
	Streams           []StreamInfo
	UnreadableStreams []UnreadableStream
}

// UnreadableStream is a stream whose consumers could not be listed.
type UnreadableStream struct {
	Stream string

	// Access is the refused permission; nil when the failure was not a permissions violation.
	Access *AccessCheck

	// Err is the failure when it was not a permissions violation.
	Err error
}
