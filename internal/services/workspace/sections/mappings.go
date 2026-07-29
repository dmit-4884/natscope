// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"context"
	"encoding/json"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"

	mappingssvc "github.com/dmit-4884/natscope/internal/services/mappings"
	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

type mappingItem struct {
	Pattern     string `json:"pattern"`
	MessageType string `json:"messageType"`
	// SourceName is the portable export-time key, resolved back to the local id on
	// import; SourceID is only a same-machine fallback.
	SourceName        string  `json:"sourceName,omitempty"`
	SourceID          string  `json:"sourceId,omitempty"`
	PinnedTag         *string `json:"pinnedTag,omitempty"`
	PinnedFingerprint *string `json:"pinnedFingerprint,omitempty"`
}

// MappingsSection exports/imports subject→message-type mappings, keyed by
// pattern; sources are resolved by name, so proto_sources must import first.
type MappingsSection struct {
	svc      mappingssvc.Service
	protoSvc protosvc.SourceManager
}

// NewMappingsSection constructs the section. protoSvc translates source id↔name
// for portable imports; nil in tests round-trips by raw source id.
func NewMappingsSection(svc mappingssvc.Service, protoSvc protosvc.SourceManager) *MappingsSection {
	return &MappingsSection{svc: svc, protoSvc: protoSvc}
}

func (s *MappingsSection) Key() string { return "mappings" }

func (s *MappingsSection) Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error) {
	all, err := s.svc.GetAll(ctx)
	if err != nil {
		return entities.WorkspaceSectionInfo{}, err
	}
	return entities.WorkspaceSectionInfo{Key: s.Key(), Title: "Subject Mappings", Count: int32(len(all))}, nil
}

func (s *MappingsSection) Export(ctx context.Context) (json.RawMessage, error) {
	all, err := s.svc.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	idToName, err := s.sourceIDToName(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]mappingItem, 0, len(all))
	for _, m := range all {
		items = append(items, mappingItem{
			Pattern:           m.Pattern,
			MessageType:       m.MessageType,
			SourceName:        idToName[m.SourceID], // "" when unresolved — falls back to SourceID on import
			SourceID:          m.SourceID,
			PinnedTag:         m.PinnedTag,
			PinnedFingerprint: m.PinnedFingerprint,
		})
	}
	return json.Marshal(newItemsPayload(items))
}

func (s *MappingsSection) Validate(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionReport, error) {
	items, err := decodeItems[mappingItem](raw)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	existing, err := s.existingPatterns(ctx)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	created, updated, deleted, conflicts := upsertPlan(
		existing,
		slices.To(items, func(i mappingItem) string { return i.Pattern }),
		strategy,
	)
	return entities.WorkspaceSectionReport{Created: created, Updated: updated, Deleted: deleted, Conflicts: conflicts}, nil
}

func (s *MappingsSection) Import(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionResult, error) {
	items, err := decodeItems[mappingItem](raw)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}
	nameToID, err := s.sourceNameToID(ctx)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}
	all, err := s.svc.GetAll(ctx)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}
	byPattern := maps.FromSliceWith(all, func(m *entities.SubjectMapping) (string, *entities.SubjectMapping) {
		return m.Pattern, m
	})

	var res entities.WorkspaceSectionResult
	if strategy == entities.WorkspaceStrategyReplace {
		for _, m := range all {
			if err := s.svc.Delete(ctx, m.Id); err != nil {
				return res, err
			}
			res.Deleted++
		}
		byPattern = map[string]*entities.SubjectMapping{}
	}

	for _, it := range items {
		resolvedSourceID := s.resolveSourceID(it, nameToID)
		if ex, ok := byPattern[it.Pattern]; ok {
			upd := &entities.SubjectMappingUpdate{
				Id:                ex.Id,
				MessageType:       ptr.Wrap(it.MessageType),
				PinnedTag:         it.PinnedTag,
				PinnedFingerprint: it.PinnedFingerprint,
			}
			// Only overwrite the source id when concrete — an empty value must NOT
			// clobber the existing source with "".
			if resolvedSourceID != "" {
				upd.SourceID = ptr.Wrap(resolvedSourceID)
			}
			if _, err := s.svc.Update(ctx, upd); err != nil {
				return res, err
			}
			res.Updated++
			continue
		}
		if _, err := s.svc.Create(ctx, &entities.SubjectMappingCreate{
			Pattern:           it.Pattern,
			MessageType:       it.MessageType,
			SourceID:          resolvedSourceID,
			PinnedTag:         it.PinnedTag,
			PinnedFingerprint: it.PinnedFingerprint,
		}); err != nil {
			return res, err
		}
		res.Created++
	}
	return res, nil
}

// resolveSourceID prefers the portable SourceName resolved against local
// sources, falling back to the raw SourceID.
func (s *MappingsSection) resolveSourceID(it mappingItem, nameToID map[string]string) string {
	if it.SourceName != "" {
		if id, ok := nameToID[it.SourceName]; ok {
			return id
		}
	}
	return it.SourceID
}

func (s *MappingsSection) sourceIDToName(ctx context.Context) (map[string]string, error) {
	srcs, err := s.listSources(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(srcs, func(src *entities.ProtoSource) (string, string) {
		return src.Id, src.Name
	}), nil
}

func (s *MappingsSection) sourceNameToID(ctx context.Context) (map[string]string, error) {
	srcs, err := s.listSources(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(srcs, func(src *entities.ProtoSource) (string, string) {
		return src.Name, src.Id
	}), nil
}

// listSources returns local proto sources, or empty when no proto service is
// wired (tests).
func (s *MappingsSection) listSources(ctx context.Context) (entities.ProtoSources, error) {
	if s.protoSvc == nil {
		return nil, nil
	}
	res, err := s.protoSvc.ListSources(ctx, &entities.ProtoSourcesList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(listAllLimit)},
	})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (s *MappingsSection) existingPatterns(ctx context.Context) (map[string]struct{}, error) {
	all, err := s.svc.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(all, func(m *entities.SubjectMapping) (string, struct{}) {
		return m.Pattern, struct{}{}
	}), nil
}
