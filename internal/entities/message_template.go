// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import "github.com/altessa-s/go-atlas/domain/converter"

// MessageTemplate is a reusable publish preset (subject, message type, JSON
// body, headers) reloadable while publishing.
type MessageTemplate struct {
	BaseEntity

	// Name is the human-readable label.
	Name string

	// Subject is the optional subject pattern (NATS wildcards * and > supported).
	Subject string

	// MessageType is the optional fully qualified protobuf message type.
	MessageType string

	// Data is the stored payload (raw JSON, may include {{helpers}} placeholders).
	Data string

	// Headers is the optional header map attached when publishing.
	Headers map[string]string

	// Wildcards holds positional resolved values for Subject's `*`/`>` slots; not
	// length-enforced (publish UI tolerates short/long).
	Wildcards []string
}

// MessageTemplateNew creates a new MessageTemplate with generated Id and
// timestamps.
func MessageTemplateNew(init ...func(*MessageTemplate)) *MessageTemplate {
	t := &MessageTemplate{
		BaseEntity: *New(),
	}
	if len(init) > 0 && init[0] != nil {
		init[0](t)
	}
	return t
}

// ApplyUpdate applies the partial update; non-nil Headers/Wildcards replace
// wholesale (nil keeps existing) since converter can't tell nil from empty.
func (t *MessageTemplate) ApplyUpdate(req *MessageTemplateUpdate) {
	if t == nil || req == nil {
		return
	}
	converter.Convert(req, t,
		converter.WithIgnoreNilValues(),
		converter.WithIgnoreFields("etag", "Headers", "Wildcards"))
	if req.Headers != nil {
		t.Headers = req.Headers
	}
	if req.Wildcards != nil {
		t.Wildcards = req.Wildcards
	}
	t.BeforeUpdate()
}

// MessageTemplates is a slice of MessageTemplate pointers.
type MessageTemplates []*MessageTemplate

// MessageTemplatesList is the listing filter for templates.
type MessageTemplatesList struct {
	ListBase
}

// MessageTemplateCreate is the create DTO for a template.
type MessageTemplateCreate struct {
	Name        string `normalize:"trim"`
	Subject     string `normalize:"trim"`
	MessageType string `normalize:"trim"`
	Data        string
	Headers     map[string]string
	Wildcards   []string
}

// MessageTemplateUpdate is the partial update DTO for a template.
type MessageTemplateUpdate struct {
	Id          string  `normalize:"trim"`
	Name        *string `normalize:"trim"`
	Subject     *string `normalize:"trim"`
	MessageType *string `normalize:"trim"`
	Data        *string
	// Headers non-nil REPLACES the stored map (empty = clear, nil = keep).
	Headers map[string]string
	// Wildcards non-nil REPLACES the stored slice (empty = clear, nil = keep).
	Wildcards []string
}
