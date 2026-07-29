// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "github.com/altessa-s/go-atlas/domain/converter"

// MappingHealth is a mapping's resolvability state against current proto
// sources/selections/descriptors.
type MappingHealth string

const (
	// MappingHealthOK resolves cleanly (source enabled, selection set, descriptor
	// compiled, type present).
	MappingHealthOK MappingHealth = "ok"

	// MappingHealthSourceMissing referenced source no longer exists.
	MappingHealthSourceMissing MappingHealth = "source_missing"

	// MappingHealthSourceDisabled referenced source is disabled.
	MappingHealthSourceDisabled MappingHealth = "source_disabled"

	// MappingHealthSelectionMissing git source has no selected tag.
	MappingHealthSelectionMissing MappingHealth = "selection_missing"

	// MappingHealthDescriptorMissing no compiled descriptor for (source, tag).
	MappingHealthDescriptorMissing MappingHealth = "descriptor_missing"

	// MappingHealthTypeMissing message type not present in the active snapshot.
	MappingHealthTypeMissing MappingHealth = "type_missing"
)

// SubjectMapping maps a NATS subject pattern to a proto message type, scoped to
// one proto source; same pattern across different sources is allowed.
type SubjectMapping struct {
	BaseEntity

	// Pattern is the NATS subject pattern (e.g., "orders.>" or "users.*").
	Pattern string

	// MessageType is the fully qualified protobuf message type (e.g.,
	// "api.v1.OrderEvent").
	MessageType string

	// SourceID is required; decode resolves descriptors only from this source's
	// active selection.
	SourceID string

	// PinnedTag overrides the source's active selection — decode historical
	// messages from an older still-compiled tag.
	PinnedTag *string

	// PinnedFingerprint pins resolution to a descriptor by content hash (survives
	// identical recompiles); takes precedence over PinnedTag.
	PinnedFingerprint *string
}

// SubjectMappingNew creates a new SubjectMapping with generated Id and
// timestamps.
func SubjectMappingNew(init ...func(*SubjectMapping)) *SubjectMapping {
	m := &SubjectMapping{
		BaseEntity: *New(),
	}

	if len(init) > 0 && init[0] != nil {
		init[0](m)
	}

	return m
}

// ApplyUpdate applies the update request to the mapping.
func (m *SubjectMapping) ApplyUpdate(req *SubjectMappingUpdate) {
	if m == nil || req == nil {
		return
	}

	converter.Convert(req, m,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("etag"))
	m.BeforeUpdate()
}

// SubjectMappings is a slice of SubjectMapping pointers.
type SubjectMappings []*SubjectMapping

// SubjectMappingsList is the listing filter for mappings.
type SubjectMappingsList struct {
	ListBase
}

// SubjectMappingCreate is the create DTO for a mapping.
type SubjectMappingCreate struct {
	Pattern           string  `normalize:"trim"`
	MessageType       string  `normalize:"trim"`
	SourceID          string  `normalize:"trim"`
	PinnedTag         *string `normalize:"trim,nil_on_empty"`
	PinnedFingerprint *string `normalize:"trim,nil_on_empty"`
}

// SubjectMappingUpdate is the update DTO for a mapping.
type SubjectMappingUpdate struct {
	Id                string  `normalize:"trim"`
	Pattern           *string `normalize:"trim"`
	MessageType       *string `normalize:"trim"`
	SourceID          *string `normalize:"trim"`
	PinnedTag         *string `normalize:"trim"`
	PinnedFingerprint *string `normalize:"trim"`
}

// SubjectMappingHealth wraps a SubjectMapping with computed health state and a
// human-readable detail.
type SubjectMappingHealth struct {
	Id     string
	Health MappingHealth
	Detail string
}

// SubjectMappingBulkSaveResult reports the outcome of a full-set replace via
// BulkSave: how many pairs were created, updated in place, or deleted.
type SubjectMappingBulkSaveResult struct {
	Created int
	Updated int
	Deleted int
}
