// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoField is a field from a proto message descriptor.
type ProtoField struct {
	Name      string
	Number    int32
	Type      string
	Label     string
	IsMessage bool
}
