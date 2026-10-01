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

	// SourceRevision is the revision of the schema the type comes from.
	SourceRevision string
}
