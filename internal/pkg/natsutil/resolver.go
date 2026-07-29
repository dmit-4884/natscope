// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package natsutil

import (
	"cmp"
	"slices"
	"strings"

	"github.com/dmit-4884/natscope/internal/entities"

	atlasslices "github.com/altessa-s/go-atlas/core/collections/slices"
)

// MappingResolver resolves NATS subjects to SubjectMapping entries deterministically.
// Exact match wins; wildcard ties break by specificity, then pattern, then CreatedAt.
type MappingResolver struct {
	exact map[string]*entities.SubjectMapping
	wild  []*entities.SubjectMapping
}

// NewMappingResolver builds a resolver from the given mappings.
// Nil entries are skipped silently. The input slice is not mutated.
func NewMappingResolver(ms entities.SubjectMappings) *MappingResolver {
	r := &MappingResolver{
		exact: make(map[string]*entities.SubjectMapping, len(ms)),
		wild:  make([]*entities.SubjectMapping, 0, len(ms)),
	}
	for _, m := range ms {
		if m == nil {
			continue
		}
		if hasWildcard(m.Pattern) {
			r.wild = append(r.wild, m)
		} else {
			// If two exact mappings share the same pattern, last write wins;
			// storage layer already guards against (Pattern, SourceID) duplicates.
			r.exact[m.Pattern] = m
		}
	}
	slices.SortStableFunc(r.wild, func(a, b *entities.SubjectMapping) int {
		if c := cmp.Compare(specificity(b.Pattern), specificity(a.Pattern)); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Pattern, b.Pattern); c != 0 {
			return c
		}
		return a.CreatedAt.Compare(b.CreatedAt)
	})
	return r
}

// Resolve returns the best-matching mapping for a subject, or nil if no mapping matches.
func (r *MappingResolver) Resolve(subject string) *entities.SubjectMapping {
	if r == nil {
		return nil
	}
	if m, ok := r.exact[subject]; ok {
		return m
	}
	for _, m := range r.wild {
		if MatchSubject(m.Pattern, subject) {
			return m
		}
	}
	return nil
}

// Mappings returns all mappings the resolver was built from (exact + wildcard).
// The returned slice is a fresh copy and may be sorted by the caller.
func (r *MappingResolver) Mappings() entities.SubjectMappings {
	if r == nil {
		return nil
	}
	out := make(entities.SubjectMappings, 0, len(r.exact)+len(r.wild))
	for _, m := range r.exact {
		out = append(out, m)
	}
	out = append(out, r.wild...)
	return out
}

// hasWildcard reports whether the NATS pattern contains a wildcard token.
func hasWildcard(pattern string) bool {
	return atlasslices.Any(strings.Split(pattern, "."), func(t string) bool {
		return t == "*" || t == ">"
	})
}

// specificity scores a pattern: literal token +10, "*" +1, ">" -5 (least specific).
func specificity(pattern string) int {
	score := 0
	for _, t := range strings.Split(pattern, ".") {
		switch t {
		case ">":
			score -= 5
		case "*":
			score++
		default:
			score += 10
		}
	}
	return score
}
