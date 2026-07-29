// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ClusterStats is stream cluster information.
type ClusterStats struct {
	Name     string
	Leader   string
	Replicas []string
}
