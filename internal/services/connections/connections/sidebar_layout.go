// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package connections

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/altessa-s/go-atlas/domain/normalizer"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
)

// GetSidebarLayout returns the connection's sidebar layout, empty when none was
// saved.
func (s *Service) GetSidebarLayout(ctx context.Context, connectionID string) (*entities.SidebarLayout, error) {
	if err := s.requireConnection(ctx, connectionID); err != nil {
		return nil, err
	}
	layout, err := s.layouts.Get(ctx, connectionID)
	if errors.Is(err, errs.ErrSidebarLayoutNotFound) {
		return &entities.SidebarLayout{ConnectionID: connectionID}, nil
	}
	return layout, err
}

// UpdateSidebarLayout replaces the sections set in `in`, cleaned by
// cleanSection, and keeps the others.
func (s *Service) UpdateSidebarLayout(ctx context.Context, in *entities.SidebarLayoutUpdate) (*entities.SidebarLayout, error) {
	_ = normalizer.Normalize(in) //nolint:errcheck // canonical: normalize tags can't fail on a well-formed DTO

	if err := s.requireConnection(ctx, in.ConnectionID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	return s.layouts.Update(ctx, in.ConnectionID, func(existing *entities.SidebarLayout) {
		for _, section := range []struct {
			dst *entities.SectionLayout
			src *entities.SectionLayout
		}{
			{dst: &existing.Streams, src: in.Streams},
			{dst: &existing.KV, src: in.KV},
			{dst: &existing.Objects, src: in.Objects},
		} {
			if section.src != nil {
				*section.dst = cleanSection(*section.src)
			}
		}
		existing.UpdatedAt = now
	})
}

// requireConnection returns errs.ErrSavedConnectionNotFound unless the
// connection is saved.
func (s *Service) requireConnection(ctx context.Context, connectionID string) error {
	ok, err := s.storage.Exists(ctx, connectionID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.ErrSavedConnectionNotFound
	}
	return nil
}

// cleanSection drops blank and repeated names; a pinned name is dropped from
// Order, since pinning already places it.
func cleanSection(in entities.SectionLayout) entities.SectionLayout {
	seen := make(map[string]struct{}, len(in.Pinned)+len(in.Order))
	keep := func(names []string) []string {
		out := make([]string, 0, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if _, dup := seen[name]; name == "" || dup {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
		return out
	}
	pinned := keep(in.Pinned)
	return entities.SectionLayout{Pinned: pinned, Order: keep(in.Order)}
}
