// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package mcptransport

import (
	"context"
	"strings"

	"github.com/altessa-s/go-atlas/core/collections/slices"

	"github.com/dmit-4884/natscope/internal/entities"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

const maxTypeSuggestions = 5

// MessageTypes lists the message types of every enabled source.
func MessageTypes(ctx context.Context, registry protosvc.Registry) ([]entities.SchemaType, error) {
	all, err := registry.ListTypes(ctx, "")
	if err != nil {
		return nil, err
	}
	var out []entities.SchemaType
	for _, t := range all {
		if t.Kind == entities.SchemaTypeMessage {
			out = append(out, t)
		}
	}
	return out, nil
}

// LookupType finds a loaded Protobuf message type by full name; sourceID disambiguates types defined by several sources.
func LookupType(ctx context.Context, registry protosvc.Registry, name, sourceID string) (entities.SchemaType, error) {
	name = strings.TrimPrefix(strings.TrimSpace(name), ".")
	all, err := MessageTypes(ctx, registry)
	if err != nil {
		return entities.SchemaType{}, err
	}
	var found []entities.SchemaType
	for _, m := range all {
		if m.FullName == name && (sourceID == "" || m.SourceID == sourceID) {
			found = append(found, m)
		}
	}
	switch len(found) {
	case 1:
		return found[0], nil
	case 0:
		return entities.SchemaType{}, Errorf("message type %q is not loaded%s", name, suggestTypes(all, name))
	default:
		return entities.SchemaType{}, Errorf("message type %q is defined by several sources (%s): pass sourceId",
			name, strings.Join(slices.To(found, func(m entities.SchemaType) string { return m.SourceID }), ", "))
	}
}

func suggestTypes(all []entities.SchemaType, name string) string {
	short := strings.ToLower(name[strings.LastIndex(name, ".")+1:])
	var similar []string
	for _, m := range all {
		if len(similar) < maxTypeSuggestions && short != "" && strings.Contains(strings.ToLower(m.FullName), short) {
			similar = append(similar, m.FullName)
		}
	}
	if len(similar) == 0 {
		return "; list_message_types shows the loaded ones"
	}
	return "; similar: " + strings.Join(similar, ", ")
}
