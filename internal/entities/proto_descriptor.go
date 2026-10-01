// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

// ProtoDescriptor is the compiled FileDescriptorSet of a source at one revision.
type ProtoDescriptor struct {
	BaseEntity

	SourceID      string
	Revision      string
	DescriptorSet []byte
	// Fingerprint is the SHA-256 of DescriptorSet.
	Fingerprint  string
	MessageTypes []string
	TargetFiles  []string
	CompiledAt   int64
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
