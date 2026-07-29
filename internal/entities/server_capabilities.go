// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ServerCapabilities describes which version-gated NATS features the
// connected server supports. Flags exist only for features the GUI exposes.
type ServerCapabilities struct {
	// ApiLevel is the JetStream API level advertised by the server
	// (0 on servers older than 2.12).
	ApiLevel int32
	// ConsumerPause reports pause/resume consumer support (NATS 2.11+, API level 1).
	ConsumerPause bool
	// MessageTtl reports per-message TTL support (NATS 2.11+, API level 1).
	MessageTtl bool
	// AtomicPublish reports atomic batch publish support (NATS 2.12+, API level 2).
	AtomicPublish bool
}
