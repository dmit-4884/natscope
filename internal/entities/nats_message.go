// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// NatsMessage is a message received from a NATS subscription.
type NatsMessage struct {
	Subject string

	Data []byte

	Header map[string][]string

	// JetStream metadata (only present for JetStream subscriptions).
	Sequence  *uint64
	Stream    string
	Timestamp time.Time
}

// DetectContentType determines if data is JSON, text, or binary.
func (m *NatsMessage) DetectContentType() ContentType {
	return DetectContentType(m.Data)
}

// Subscription is an active NATS subscription.
type Subscription interface {
	Unsubscribe() error
}

// MessageHandler is a callback function for handling incoming messages.
type MessageHandler func(*NatsMessage)
