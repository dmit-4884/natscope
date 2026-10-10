// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package entities

// ClusterStats is stream cluster information.
type ClusterStats struct {
	Name     string
	Leader   string
	Replicas []string
}
