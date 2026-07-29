// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// ConsumerStats is a stat snapshot for one JetStream consumer.
type ConsumerStats struct {
	// Basic info
	Name    string
	Stream  string
	Created time.Time

	// Consumer state
	NumPending     uint64
	NumAckPending  int
	NumRedelivered int
	NumWaiting     int
	Delivered      SequenceInfo
	AckFloor       SequenceInfo

	// Consumer type (PushBound maps directly from jetstream.ConsumerInfo.PushBound)
	PushBound bool

	// Configuration - Delivery
	DeliverPolicy  DeliverPolicy
	OptStartSeq    uint64
	OptStartTime   time.Time // zero means not set
	FilterSubject  string
	FilterSubjects []string

	// Configuration - Acknowledgement
	AckPolicy     AckPolicy
	AckWait       time.Duration
	MaxAckPending int

	// Configuration - Redelivery
	MaxDeliver   int
	ReplayPolicy ReplayPolicy
	BackOff      []time.Duration

	// Configuration - Rate limit
	RateLimit uint64

	// Configuration - Pull specific
	MaxWaiting         int
	MaxRequestBatch    int
	MaxRequestExpires  time.Duration
	MaxRequestMaxBytes int

	// Configuration - Other
	Durable           string
	Description       string
	SampleFrequency   string
	InactiveThreshold time.Duration
	HeadersOnly       bool
	Replicas          int
	MemoryStorage     bool
	Metadata          map[string]string

	// Cluster info (reuses ClusterInfo — same type as StreamInfo/ConsumerInfo)
	Cluster *ClusterInfo
}
