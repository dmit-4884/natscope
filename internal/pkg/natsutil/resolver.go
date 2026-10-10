// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package natsutil

import (
	"cmp"
	"slices"
	"strings"

	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/data/cache/lru"

	"github.com/dmit-4884/natscope/internal/entities"
)

// maxResolverCacheEntries bounds the wildcard-resolution cache; the least recently used subjects are evicted past it.
const maxResolverCacheEntries = 10000

// MappingResolver resolves NATS subjects to SubjectMapping entries deterministically.
// Exact match wins; wildcard ties break by specificity, then pattern, then CreatedAt.
// A resolver is rebuilt on every mapping mutation, so its cache needs no invalidation.
type MappingResolver struct {
	exact map[string]*entities.SubjectMapping
	wild  []*entities.SubjectMapping

	// cache memoizes wildcard Resolve results per subject, nil included.
	cache *lru.ShardedCache[string, *entities.SubjectMapping]
}

// NewMappingResolver builds a resolver from the given mappings.
// Nil entries are skipped silently. The input slice is not mutated.
func NewMappingResolver(ms entities.SubjectMappings) *MappingResolver {
	r := &MappingResolver{
		exact: make(map[string]*entities.SubjectMapping, len(ms)),
		wild:  make([]*entities.SubjectMapping, 0, len(ms)),
		cache: panics.MustResult(lru.NewShardedCache[string, *entities.SubjectMapping](maxResolverCacheEntries)),
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
// Wildcard lookups are memoized per subject.
func (r *MappingResolver) Resolve(subject string) *entities.SubjectMapping {
	if r == nil {
		return nil
	}
	if m, ok := r.exact[subject]; ok {
		return m
	}
	if len(r.wild) == 0 {
		return nil
	}

	if m, ok := r.cache.Get(subject); ok {
		return m
	}

	subjectTokens := strings.Split(subject, ".")
	var found *entities.SubjectMapping
	for _, m := range r.wild {
		if matchSubjectTokens(m.Pattern, subjectTokens) {
			found = m
			break
		}
	}

	r.cache.Put(subject, found)
	return found
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
	return slices.ContainsFunc(strings.Split(pattern, "."), func(t string) bool {
		return t == "*" || t == ">"
	})
}

// specificity scores a pattern: literal token +10, "*" +1, ">" -5 (least specific).
func specificity(pattern string) int {
	score := 0
	for t := range strings.SplitSeq(pattern, ".") {
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
