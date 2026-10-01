// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package sections

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/altessa-s/go-atlas/core/collections/maps"
	"github.com/altessa-s/go-atlas/core/collections/slices"
	"github.com/altessa-s/go-atlas/core/types/ptr"

	"github.com/dmit-4884/natscope/internal/entities"

	protosvc "github.com/dmit-4884/natscope/internal/services/proto"
)

// protoSourceItem is the NON-SECRET projection of a proto source — git auth
// Token deliberately absent.
type protoSourceItem struct {
	Name            string      `json:"name"`
	SourceType      string      `json:"sourceType"`
	Repository      string      `json:"repository,omitempty"`
	LocalPath       *string     `json:"localPath,omitempty"`
	WatcherEnabled  bool        `json:"watcherEnabled,omitempty"`
	ImportRoots     []string    `json:"importRoots,omitempty"`
	ExcludePrefixes []string    `json:"excludePrefixes,omitempty"`
	Ref             string      `json:"ref,omitempty"`
	Upload          *uploadItem `json:"upload,omitempty"`
}

type uploadItem struct {
	Files         []uploadFileItem `json:"files,omitempty"`
	DescriptorSet []byte           `json:"descriptorSet,omitempty"`
}

type uploadFileItem struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// ProtoSourcesSection exports/imports proto sources WITHOUT git tokens; merge
// keeps existing names, creates new ones.
type ProtoSourcesSection struct {
	svc protosvc.SourceManager
}

// NewProtoSourcesSection constructs the proto-sources workspace section.
func NewProtoSourcesSection(svc protosvc.SourceManager) *ProtoSourcesSection {
	return &ProtoSourcesSection{svc: svc}
}

func (s *ProtoSourcesSection) Key() string { return "proto_sources" }

func (s *ProtoSourcesSection) Describe(ctx context.Context) (entities.WorkspaceSectionInfo, error) {
	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionInfo{}, err
	}
	return entities.WorkspaceSectionInfo{Key: s.Key(), Title: "Proto Sources (no tokens)", Count: int32(len(all))}, nil
}

func (s *ProtoSourcesSection) Export(ctx context.Context) (json.RawMessage, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]protoSourceItem, 0, len(all))
	for _, src := range all {
		item := redactProtoSource(src)
		if src.SourceType == entities.SourceTypeUpload && src.ActiveSchema != nil {
			upload, uErr := s.svc.UploadedSchema(ctx, src.Id)
			if uErr != nil {
				return nil, uErr
			}
			item.Upload = &uploadItem{
				DescriptorSet: upload.DescriptorSet,
				Files: slices.To(upload.Files, func(f entities.ProtoFileEntry) uploadFileItem {
					return uploadFileItem{Path: f.Path, Content: f.Content}
				}),
			}
		}
		items = append(items, item)
	}
	return json.Marshal(newItemsPayload(items))
}

func (s *ProtoSourcesSection) Validate(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionReport, error) {
	items, err := decodeItems[protoSourceItem](raw)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	existing, err := s.existingNames(ctx)
	if err != nil {
		return entities.WorkspaceSectionReport{}, err
	}
	created, deleted, conflicts := createOnlyPlan(
		existing,
		slices.To(items, func(i protoSourceItem) string { return i.Name }),
		strategy,
	)
	rep := entities.WorkspaceSectionReport{Created: created, Deleted: deleted, Conflicts: conflicts}
	if hasGitSource(items) {
		rep.Warnings = append(rep.Warnings, "git sources are imported without tokens — set them after import")
	}
	if deleted > 0 {
		rep.Warnings = append(rep.Warnings,
			"REPLACE will permanently delete the existing proto sources AND their stored git tokens")
	}
	return rep, nil
}

func (s *ProtoSourcesSection) Import(
	ctx context.Context,
	raw json.RawMessage,
	strategy entities.WorkspaceStrategy,
) (entities.WorkspaceSectionResult, error) {
	items, err := decodeItems[protoSourceItem](raw)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}
	all, err := s.all(ctx)
	if err != nil {
		return entities.WorkspaceSectionResult{}, err
	}

	var res entities.WorkspaceSectionResult
	if strategy == entities.WorkspaceStrategyReplace {
		// Delete before create: the active-name unique index rejects a create that
		// collides with a not-yet-deleted record.
		for _, src := range all {
			if dErr := s.svc.DeleteSource(ctx, src.Id); dErr != nil {
				return res, dErr
			}
			res.Deleted++
		}
		for _, it := range items {
			if cErr := s.create(ctx, it, &res); cErr != nil {
				return res, cErr
			}
		}
		if hasGitSource(items) {
			res.Warnings = append(res.Warnings, "git sources imported without tokens — set them before fetching")
		}
		return res, nil
	}

	existing := maps.FromSliceWith(all, func(src *entities.ProtoSource) (string, struct{}) {
		return src.Name, struct{}{}
	})
	for _, it := range items {
		if _, ok := existing[it.Name]; ok {
			continue
		}
		if cErr := s.create(ctx, it, &res); cErr != nil {
			return res, cErr
		}
	}
	if hasGitSource(items) {
		res.Warnings = append(res.Warnings, "git sources imported without tokens — set them before fetching")
	}
	return res, nil
}

func (s *ProtoSourcesSection) create(ctx context.Context, it protoSourceItem, res *entities.WorkspaceSectionResult) error {
	src, err := s.svc.CreateSource(ctx, toProtoSourceCreate(it))
	if err != nil {
		return err
	}
	res.Created++
	switch {
	case it.Ref != "":
		_, outcome, err := s.svc.SelectRef(ctx, src.Id, it.Ref)
		switch {
		case err != nil:
			res.Warnings = append(res.Warnings, fmt.Sprintf("source %q: ref %q not selected: %v", it.Name, it.Ref, err))
		case !outcome.Valid:
			res.Warnings = append(res.Warnings, fmt.Sprintf("source %q: ref %q does not compile", it.Name, it.Ref))
		}
	case it.Upload != nil:
		_, outcome, err := s.svc.UploadSchema(ctx, src.Id, entities.SchemaUpload{
			DescriptorSet: it.Upload.DescriptorSet,
			Files: slices.To(it.Upload.Files, func(f uploadFileItem) entities.ProtoFileEntry {
				return entities.ProtoFileEntry{Path: f.Path, Content: f.Content}
			}),
		})
		switch {
		case err != nil:
			res.Warnings = append(res.Warnings, fmt.Sprintf("source %q: upload not restored: %v", it.Name, err))
		case !outcome.Valid:
			res.Warnings = append(res.Warnings, fmt.Sprintf("source %q: uploaded schema does not compile", it.Name))
		}
	}
	return nil
}

func redactProtoSource(src *entities.ProtoSource) protoSourceItem {
	var ref string
	if src.SelectedRef != nil {
		ref = src.SelectedRef.Name
	}
	return protoSourceItem{
		Ref:             ref,
		Name:            src.Name,
		SourceType:      string(src.SourceType),
		Repository:      src.Repository,
		LocalPath:       src.LocalPath,
		WatcherEnabled:  src.WatcherEnabled,
		ImportRoots:     src.ImportRoots,
		ExcludePrefixes: src.ExcludePrefixes,
	}
}

func toProtoSourceCreate(it protoSourceItem) *entities.ProtoSourceCreate {
	return &entities.ProtoSourceCreate{
		Name:            it.Name,
		SourceType:      entities.SourceType(it.SourceType),
		Repository:      it.Repository,
		LocalPath:       it.LocalPath,
		WatcherEnabled:  ptr.Wrap(it.WatcherEnabled),
		ImportRoots:     it.ImportRoots,
		ExcludePrefixes: it.ExcludePrefixes,
	}
}

func hasGitSource(items []protoSourceItem) bool {
	return slices.Any(items, func(it protoSourceItem) bool {
		return it.SourceType == string(entities.SourceTypeGit)
	})
}

func (s *ProtoSourcesSection) all(ctx context.Context) (entities.ProtoSources, error) {
	res, err := s.svc.ListSources(ctx, &entities.ProtoSourcesList{
		ListBase: entities.ListBase{Limit: ptr.Wrap(listAllLimit)},
	})
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func (s *ProtoSourcesSection) existingNames(ctx context.Context) (map[string]struct{}, error) {
	all, err := s.all(ctx)
	if err != nil {
		return nil, err
	}
	return maps.FromSliceWith(all, func(src *entities.ProtoSource) (string, struct{}) {
		return src.Name, struct{}{}
	}), nil
}
