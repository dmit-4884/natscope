// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ClusterInfo is stream cluster information.
type ClusterInfo struct {
	// Name is the cluster name.
	Name string

	// RaftGroup is the Raft group name.
	RaftGroup string

	// Leader is the current leader server name.
	Leader string

	// Replicas is the peer state for each replica.
	Replicas []*PeerInfo
}
