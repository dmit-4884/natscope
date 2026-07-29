// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// ConsumerConfig is the full option set for a JetStream consumer.
type ConsumerConfig struct {
	// Name is the consumer name.
	Name string

	// Durable is the durable name for the consumer.
	Durable string

	// Description is the consumer description.
	Description string

	// DeliverPolicy determines where to start delivering messages.
	DeliverPolicy DeliverPolicy

	// OptStartSeq is the starting sequence for deliver_by_start_sequence.
	OptStartSeq uint64

	// OptStartTime is the starting time for deliver_by_start_time.
	OptStartTime string

	// AckPolicy is the acknowledgment policy.
	AckPolicy AckPolicy

	// AckWait is the acknowledgment wait time.
	AckWait time.Duration

	// MaxDeliver is the maximum delivery attempts.
	MaxDeliver int

	// BackOff is the backoff intervals.
	BackOff []time.Duration

	// FilterSubject is the subject filter (single subject).
	FilterSubject string

	// FilterSubjects is the list of subject filters (multiple subjects, NATS 2.10+).
	FilterSubjects []string

	// ReplayPolicy is the replay policy.
	ReplayPolicy ReplayPolicy

	// RateLimit is the rate limit in bits per second.
	RateLimit uint64

	// SampleFrequency is the sampling frequency for observability.
	SampleFrequency string

	// MaxWaiting is the maximum pending pull requests.
	MaxWaiting int

	// MaxAckPending is the maximum pending acknowledgments.
	MaxAckPending int

	// FlowControl enables flow control.
	FlowControl bool

	// IdleHeartbeat is the idle heartbeat interval.
	IdleHeartbeat time.Duration

	// HeadersOnly delivers only headers, not message bodies.
	HeadersOnly bool

	// MaxRequestBatch is the maximum batch size for pull requests.
	MaxRequestBatch int

	// MaxRequestExpires is the maximum pull request expiration.
	MaxRequestExpires time.Duration

	// InactiveThreshold is the inactive threshold.
	InactiveThreshold time.Duration

	// Replicas is the number of replicas.
	Replicas int

	// MemoryStorage uses memory storage instead of file.
	MemoryStorage bool

	// MaxRequestMaxBytes is the maximum bytes for pull requests.
	MaxRequestMaxBytes int

	// Metadata is custom key-value metadata.
	Metadata map[string]string

	// DeliverSubject is the subject for push delivery.
	DeliverSubject string

	// DeliverGroup is the queue group for load balancing.
	DeliverGroup string
}
