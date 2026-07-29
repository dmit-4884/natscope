// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// ConsumerInfo is the server-reported state of a JetStream consumer.
type ConsumerInfo struct {
	// Name is the consumer name.
	Name string

	// Stream is the stream this consumer belongs to.
	Stream string

	// Config is the consumer configuration.
	Config *ConsumerConfig

	// Created is when the consumer was created.
	Created *time.Time

	// Delivered is the delivered sequence info.
	Delivered SequenceInfo

	// AckFloor is the acknowledged sequence floor.
	AckFloor SequenceInfo

	// NumPending is the number of pending messages.
	NumPending uint64

	// NumAckPending is the number of pending acknowledgments.
	NumAckPending int

	// NumRedelivered is the number of redelivered messages.
	NumRedelivered int

	// NumWaiting is the number of waiting pull requests.
	NumWaiting int

	// PushBound indicates if this is a push-based consumer.
	PushBound bool

	// Cluster is the cluster info; nil if not clustered.
	Cluster *ClusterInfo

	// TimeStamp is the timestamp when this info was fetched.
	TimeStamp *time.Time

	// Raw is the raw JSON representation of the original jetstream.ConsumerInfo.
	Raw string
}
