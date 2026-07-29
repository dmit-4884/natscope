// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

// Package workspace implements the domain-agnostic coordinator: it iterates the
// Section registry to export, validate, and import.
package workspace

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"

	workspacesvc "github.com/dmit-4884/natscope/internal/services/workspace"
)

// Service is the registry-driven workspace coordinator.
type Service struct {
	sections map[string]workspacesvc.Section
	order    []string // section keys, sorted for stable output
}

// New builds the service from registered sections; duplicate keys resolve
// last-wins, keys sorted for deterministic output.
func New(sections []workspacesvc.Section) *Service {
	m := make(map[string]workspacesvc.Section, len(sections))
	for _, s := range sections {
		if s == nil {
			continue
		}
		m[s.Key()] = s
	}
	order := make([]string, 0, len(m))
	for k := range m {
		order = append(order, k)
	}
	sort.Strings(order)
	return &Service{sections: m, order: order}
}

// ListSections returns each section's self-description in key order; a failed
// Describe yields a zero count rather than aborting.
func (s *Service) ListSections(ctx context.Context) ([]entities.WorkspaceSectionInfo, error) {
	out := make([]entities.WorkspaceSectionInfo, 0, len(s.order))
	for _, k := range s.order {
		info, err := s.sections[k].Describe(ctx)
		if err != nil {
			out = append(out, entities.WorkspaceSectionInfo{Key: k, Title: k, Count: 0})
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

// Export builds a workspace file from the requested keys (empty = all).
func (s *Service) Export(ctx context.Context, keys []string) ([]byte, error) {
	selected, err := s.selectExportKeys(keys)
	if err != nil {
		return nil, err
	}
	file := entities.WorkspaceFile{
		Version:    entities.WorkspaceFileVersion,
		ExportedAt: time.Now().UnixMilli(),
		Sections:   make(map[string]json.RawMessage, len(selected)),
	}
	for _, k := range selected {
		raw, err := s.sections[k].Export(ctx)
		if err != nil {
			return nil, err
		}
		file.Sections[k] = raw
	}
	return json.MarshalIndent(file, "", "  ")
}

// Validate dry-runs an import.
func (s *Service) Validate(
	ctx context.Context,
	payload []byte,
	keys []string,
	strategy entities.WorkspaceStrategy,
) ([]entities.WorkspaceSectionReport, error) {
	file, selected, err := s.parseAndSelect(payload, keys)
	if err != nil {
		return nil, err
	}
	selected = orderForApply(selected)
	reports := make([]entities.WorkspaceSectionReport, 0, len(selected))
	for _, k := range selected {
		sec, ok := s.sections[k]
		if !ok {
			reports = append(reports, unknownReport(k))
			continue
		}
		rep, vErr := sec.Validate(ctx, file.Sections[k], normalizeStrategy(strategy))
		if vErr != nil {
			return nil, vErr
		}
		rep.Key = k
		reports = append(reports, rep)
	}
	return reports, nil
}

// Import applies the requested keys. A section failure is recorded as a
// warning rather than aborting; context cancellation is the exception.
func (s *Service) Import(
	ctx context.Context,
	payload []byte,
	keys []string,
	strategy entities.WorkspaceStrategy,
) ([]entities.WorkspaceSectionResult, error) {
	file, selected, err := s.parseAndSelect(payload, keys)
	if err != nil {
		return nil, err
	}
	selected = orderForApply(selected)
	results := make([]entities.WorkspaceSectionResult, 0, len(selected))
	for _, k := range selected {
		sec, ok := s.sections[k]
		if !ok {
			results = append(results, entities.WorkspaceSectionResult{
				Key:      k,
				Warnings: []string{"unknown section, skipped"},
			})
			continue
		}
		res, iErr := sec.Import(ctx, file.Sections[k], normalizeStrategy(strategy))
		res.Key = k
		if iErr != nil {
			// Canceled/expired context aborts the run, surfacing as the RPC error;
			// already-applied results returned alongside.
			if ctx.Err() != nil || errors.Is(iErr, context.Canceled) || errors.Is(iErr, context.DeadlineExceeded) {
				results = append(results, res)
				return results, iErr
			}
			res.Warnings = append(res.Warnings, "import failed: "+iErr.Error())
		}
		results = append(results, res)
	}
	return results, nil
}

// selectExportKeys returns the requested registered keys sorted (empty = all);
// an unregistered requested key is a hard error.
func (s *Service) selectExportKeys(keys []string) ([]string, error) {
	if len(keys) == 0 {
		return s.order, nil
	}
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if _, ok := s.sections[k]; !ok {
			return nil, errs.ErrWorkspaceUnknownSection
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

// parseAndSelect parses the file and returns the requested keys intersected
// with its sections (empty = all), sorted.
func (s *Service) parseAndSelect(payload []byte, keys []string) (entities.WorkspaceFile, []string, error) {
	var file entities.WorkspaceFile
	if err := json.Unmarshal(payload, &file); err != nil || file.Sections == nil {
		return entities.WorkspaceFile{}, nil, errs.ErrWorkspaceInvalidFile
	}
	// A newer-than-this-build file is rejected rather than parsed into a partial
	// shape.
	if file.Version > entities.WorkspaceFileVersion {
		return entities.WorkspaceFile{}, nil, errs.ErrWorkspaceInvalidFile
	}

	var selected []string
	if len(keys) == 0 {
		for k := range file.Sections {
			selected = append(selected, k)
		}
	} else {
		for _, k := range keys {
			if _, ok := file.Sections[k]; ok {
				selected = append(selected, k)
			}
		}
	}
	sort.Strings(selected)
	return file, selected, nil
}

func normalizeStrategy(s entities.WorkspaceStrategy) entities.WorkspaceStrategy {
	if s == entities.WorkspaceStrategyReplace {
		return entities.WorkspaceStrategyReplace
	}
	return entities.WorkspaceStrategyMerge
}

func unknownReport(key string) entities.WorkspaceSectionReport {
	return entities.WorkspaceSectionReport{
		Key:      key,
		Unknown:  true,
		Warnings: []string{"unknown section, skipped"},
	}
}

// orderForApply sorts keys into dependency-safe order then by key:
// proto_sources must apply before mappings (which resolve source names→ids).
func orderForApply(keys []string) []string {
	out := append([]string(nil), keys...)
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := importPriority(out[i]), importPriority(out[j])
		if pi != pj {
			return pi < pj
		}
		return out[i] < out[j]
	})
	return out
}

// Apply-order priorities (lower runs first): proto_sources before mappings;
// everything else sits between them.
const (
	priorityProtoSources = 0
	priorityDefault      = 10
	priorityMappings     = 20
)

// importPriority orders proto_sources before mappings; all else sits between.
func importPriority(key string) int {
	switch key {
	case "proto_sources":
		return priorityProtoSources
	case "mappings":
		return priorityMappings
	default:
		return priorityDefault
	}
}
