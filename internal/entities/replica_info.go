// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "time"

// PeerInfo is cluster peer information for a stream or consumer.
type PeerInfo struct {
	// Name is the peer server name.
	Name string

	// Current indicates if the peer is current with the leader.
	Current bool

	// Offline indicates if the peer is offline.
	Offline bool

	// Active is the time since last activity.
	Active time.Duration

	// Lag is the number of operations behind the leader.
	Lag uint64
}
