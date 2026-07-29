// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// StreamInfo is the current config, state, and cluster info for a stream.
type StreamInfo struct {
	// Config is the stream configuration.
	Config StreamConfig

	// Created is when the stream was created.
	Created time.Time

	// State is the stream's current counters; nil in summary views.
	State *StreamState

	// Cluster is the cluster info; nil for standalone deployments.
	Cluster *ClusterInfo

	// TimeStamp is the timestamp when this info was fetched.
	TimeStamp *time.Time

	// Raw is the raw JSON representation of the original jetstream.StreamInfo.
	Raw string
}
