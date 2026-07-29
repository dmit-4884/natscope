// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoDescriptor holds the compiled, serialized FileDescriptorSet for a
// source+tag (lazy-loading entity).
type ProtoDescriptor struct {
	BaseEntity

	SourceID string

	// Tag is the Git tag (e.g., "v0.72.0").
	Tag string

	// DescriptorSet is the serialized FileDescriptorSet (protobuf binary).
	DescriptorSet []byte

	// MessageTypes lists fully qualified message names for quick lookup.
	MessageTypes []string

	// CompiledAt is the Unix timestamp when descriptors were compiled.
	CompiledAt int64
}

// ProtoDescriptorNew creates a new ProtoDescriptor with generated Id and
// timestamps.
func ProtoDescriptorNew(init ...func(*ProtoDescriptor)) *ProtoDescriptor {
	d := &ProtoDescriptor{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](d)
	}

	return d
}

// ProtoDescriptors is a slice of ProtoDescriptor pointers.
type ProtoDescriptors []*ProtoDescriptor

// ProtoDescriptorsList is the listing criteria for descriptors.
type ProtoDescriptorsList struct {
	ListBase
	SourceID *string
	Tag      *string
}
