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

	templatessvc "github.com/dmit-4884/natscope/internal/services/templates"
)

// listAllLimit pulls a whole section in one List call (local stores hold
// human-scale counts).
const listAllLimit int64 = 1_000_000

type templateItem struct {
	Name        string            `json:"name"`
	Subject     string            `json:"subject"`
	MessageType string            `json:"messageType"`
	Data        string            `json:"data"`
	Headers     map[string]string `json:"headers,omitempty"`
	Wildcards   []string          `json:"wildcards,omitempty"`
}

// TemplatesSection exports/imports publish templates, keyed by name. No
// secrets.
type TemplatesSection struct {
	svc templatessvc.Service
}

// NewTemplatesSection constructs the templates workspace section.
func NewTemplatesSection(svc templatessvc.Service) *TemplatesSection {
	return &TemplatesSection{svc: svc}
}

func (s *TemplatesSection) Key() string { return "templates" }

func (s *TemplatesSection) Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error) {
	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionInfo{}, err
	}
	return entities.WorkspaceSectionInfo{Key: s.Key(), Title: "Publish Templates", Count: int32(len(all))}, nil
}

func (s *TemplatesSection) Export(ctx context.Context) (json.RawMessage, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	items := slices.To(all, func(t *entities.MessageTemplate) templateItem {
		return templateItem{
			Name:        t.Name,
			Subject:     t.Subject,
			MessageType: t.MessageType,
			Data:        t.Data,
			Headers:     t.Headers,
			Wildcards:   t.Wildcards,
		}
	})
	return json.Marshal(newItemsPayload(items))
}

func (s *TemplatesSection) Validate(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionReport, error) {
	items, err := decodeItems[templateItem](raw)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	existing, err := s.existingNames(ctx)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	created, updated, deleted, conflicts := upsertPlan(
		existing,
		slices.To(items, func(i templateItem) string { return i.Name }),
		strategy,
	)
	return entities.WorkspaceSectionReport{Created: created, Updated: updated, Deleted: deleted, Conflicts: conflicts}, nil
}

func (s *TemplatesSection) Import(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionResult, error) {
	items, err := decodeItems[templateItem](raw)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}

	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}

	var res entities.WorkspaceSectionResult
	if strategy == entities.WorkspaceStrategyReplace {
		// Create first, then delete pre-existing by id (not DeleteAll, which
		// would wipe the new rows); a mid-import failure leaves originals intact.
		for _, it := range items {
			if _, cErr := s.svc.Create(ctx, toTemplateCreate(it)); cErr != nil {
				return res, cErr
			}
			res.Created++
		}
		for _, t := range all {
			if dErr := s.svc.Delete(ctx, t.Id); dErr != nil {
				return res, dErr
			}
			res.Deleted++
		}
		return res, nil
	}

	// Merge: upsert by name, updating existing IN PLACE (preserves id/CreatedAt)
	// rather than delete-and-recreate.
	byName := maps.FromSliceWith(all, func(t *entities.MessageTemplate) (string, *entities.MessageTemplate) {
		return t.Name, t
	})
	for _, it := range items {
		if ex, ok := byName[it.Name]; ok {
			if _, uErr := s.svc.Update(ctx, toTemplateUpdate(ex.Id, it)); uErr != nil {
				return res, uErr
			}
			res.Updated++
			continue
		}
		if _, cErr := s.svc.Create(ctx, toTemplateCreate(it)); cErr != nil {
			return res, cErr
		}
		res.Created++
	}
	return res, nil
}

func toTemplateCreate(it templateItem) *entities.MessageTemplateCreate {
	return &entities.MessageTemplateCreate{
		Name:        it.Name,
		Subject:     it.Subject,
		MessageType: it.MessageType,
		Data:        it.Data,
		Headers:     it.Headers,
		Wildcards:   it.Wildcards,
	}
}

// toTemplateUpdate builds an in-place merge update, preserving id/CreatedAt
// while replacing editable fields.
func toTemplateUpdate(id string, it templateItem) *entities.MessageTemplateUpdate {
	return &entities.MessageTemplateUpdate{
		Id:          id,
		Name:        &it.Name,
		Subject:     &it.Subject,
		MessageType: &it.MessageType,
		Data:        &it.Data,
		Headers:     it.Headers,
		Wildcards:   it.Wildcards,
	}
}

func (s *TemplatesSection) all(ctx context.Context) (entities.MessageTemplates, error) {
	res, err := s.svc.List(ctx, &entities.MessageTemplatesList{ListBase: entities.ListBase{Limit: ptr.Wrap(listAllLimit)}})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (s *TemplatesSection) existingNames(ctx context.Context) (map[string]struct{}, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(all, func(t *entities.MessageTemplate) (string, struct{}) {
		return t.Name, struct{}{}
	}), nil
}
