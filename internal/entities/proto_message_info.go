// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoMessageInfo describes a protobuf message; two sources may share a
// FullName, so disambiguate by SourceID.
type ProtoMessageInfo struct {
	FullName  string
	ProtoFile string
	Package   string
	Fields    []*ProtoField

	SourceID string

	// SourceTag is the active tag of the source ("local" for local sources, e.g.
	// "v1.0.0" for git).
	SourceTag string
}
