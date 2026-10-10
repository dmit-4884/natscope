// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

import "time"

// StreamStats is a stat snapshot for one stream.
type StreamStats struct {
	Name          string
	Messages      uint64
	Bytes         uint64
	FirstSeq      uint64
	LastSeq       uint64
	ConsumerCount int
	Subjects      map[string]uint64
	Created       time.Time
	Cluster       *ClusterStats
}
