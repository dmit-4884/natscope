// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package sections

import (
	"context"
	"encoding/json"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/domain/converter"

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
		return *converter.Convert(t, &templateItem{})
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
			if _, cErr := s.svc.Create(ctx, converter.Convert(&it, &entities.MessageTemplateCreate{})); cErr != nil {
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
	if byName == nil {
		byName = make(map[string]*entities.MessageTemplate, len(items))
	}
	for _, it := range items {
		if ex, ok := byName[it.Name]; ok {
			upd, uErr := s.svc.Update(ctx, toTemplateUpdate(ex.Id, &it))
			if uErr != nil {
				return res, uErr
			}
			// Same-name items in one file resolve last-wins.
			byName[it.Name] = upd
			res.Updated++
			continue
		}
		created, cErr := s.svc.Create(ctx, converter.Convert(&it, &entities.MessageTemplateCreate{}))
		if cErr != nil {
			return res, cErr
		}
		byName[it.Name] = created
		res.Created++
	}
	return res, nil
}

// toTemplateUpdate builds an in-place merge update, preserving id/CreatedAt
// while replacing editable fields.
func toTemplateUpdate(id string, it *templateItem) *entities.MessageTemplateUpdate {
	upd := converter.Convert(it, &entities.MessageTemplateUpdate{})
	upd.Id = id
	return upd
}

func (s *TemplatesSection) all(ctx context.Context) (entities.MessageTemplates, error) {
	res, err := s.svc.List(ctx, &entities.MessageTemplatesList{Limit: new(listAllLimit)})
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
