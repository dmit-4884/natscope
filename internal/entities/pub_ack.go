// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// PubAck is a JetStream publish acknowledgment.
type PubAck struct {
	Stream string

	// Sequence is the sequence number assigned to the message.
	Sequence uint64

	// Domain is the JetStream domain (if applicable).
	Domain string

	// Duplicate indicates a duplicate message.
	Duplicate bool
}
