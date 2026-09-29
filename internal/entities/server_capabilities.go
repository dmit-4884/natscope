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
	// MessageTtl reports per-message TTL and subject delete marker support (NATS 2.11+, API level 1).
	MessageTtl bool
	// AtomicPublish reports atomic batch publish support (NATS 2.12+, API level 2).
	AtomicPublish bool
	// PriorityGroups reports pinned-client/overflow priority groups and unpin support (NATS 2.11+, API level 1).
	PriorityGroups bool
	// MsgCounters reports counter stream support (NATS 2.12+, API level 2).
	MsgCounters bool
	// MsgSchedules reports single delayed message schedule support (NATS 2.12+, API level 2).
	MsgSchedules bool
	// PriorityPrioritized reports the prioritized priority policy (NATS 2.12+, API level 2).
	PriorityPrioritized bool
	// AsyncPersist reports the async stream persist mode (NATS 2.12+, API level 2).
	AsyncPersist bool
	// ConsumerReset reports consumer reset support (NATS 2.14+, API level 4).
	ConsumerReset bool
	// CronSchedules reports repeating and cron message schedule support (NATS 2.14+, API level 4).
	CronSchedules bool
	// BatchPublish reports fast-ingest batch publish support (NATS 2.14+, API level 4).
	BatchPublish bool
}
