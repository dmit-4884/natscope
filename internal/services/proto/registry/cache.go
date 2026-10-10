// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/altessa-s/go-atlas/core/runtime/panics"
	"github.com/altessa-s/go-atlas/data/cache/lru"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// DescriptorsLookup is the minimal storage contract the cache needs, defined
// locally to keep this package free of storage-layer imports.
type DescriptorsLookup interface {
	GetBySourceRevision(ctx context.Context, sourceID, revision string) (*entities.ProtoDescriptor, error)
	GetByFingerprint(ctx context.Context, sourceID, fingerprint string) (*entities.ProtoDescriptor, error)
}

// maxCachedSnapshots bounds the cache. Snapshots hold parsed descriptor sets,
// and nothing evicts them on its own: a long session that browses many
// source/tag pairs would otherwise grow without limit.
const maxCachedSnapshots = 32

// Cache is a thread-safe per-snapshot registry cache that keeps the most
// recently used snapshots; concurrent lookups of a cold key share one parse.
type Cache struct {
	descriptors DescriptorsLookup
	entries     *lru.Cache[string, *Snapshot]
}

// NewCache constructs a new Cache wrapping the given descriptor storage.
func NewCache(descriptors DescriptorsLookup) *Cache {
	return &Cache{
		descriptors: descriptors,
		entries:     panics.MustResult(lru.NewCache[string, *Snapshot](maxCachedSnapshots)),
	}
}

// GetOrBuild returns the cached snapshot for (sourceID, revision), parsing from
// storage on miss; concurrent calls share one parse.
func (c *Cache) GetOrBuild(ctx context.Context, sourceID, revision string) (*Snapshot, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceNotFound
	}
	if revision == "" {
		return nil, errs.ErrMappingDescriptorMissing
	}

	return c.entries.GetOrCompute(ctx, snapshotKey(sourceID, revision), func(ctx context.Context) (*Snapshot, error) {
		d, err := c.descriptors.GetBySourceRevision(ctx, sourceID, revision)
		if err != nil {
			if errors.Is(err, errs.ErrProtoDescriptorNotFound) {
				return nil, errs.ErrMappingDescriptorMissing
			}
			return nil, coreerrs.Wrap(err, "registry: load descriptor")
		}
		if len(d.DescriptorSet) == 0 {
			return nil, errs.ErrMappingDescriptorMissing
		}

		schema, err := protoutils.ParseSchema(d.DescriptorSet)
		if err != nil {
			return nil, coreerrs.Wrap(err, "registry: parse descriptor")
		}

		return &Snapshot{
			SourceID:    sourceID,
			Revision:    revision,
			Descriptor:  d,
			Schema:      schema,
			ParsedAt:    time.Now(),
			Fingerprint: d.Fingerprint,
		}, nil
	})
}

// GetByFingerprint returns the cached snapshot for a content hash, scoped to
// sourceID — required, since two sources can compile to identical bytes.
func (c *Cache) GetByFingerprint(ctx context.Context, sourceID, fingerprint string) (*Snapshot, error) {
	if fingerprint == "" {
		return nil, errs.ErrMappingDescriptorMissing
	}
	if sourceID == "" {
		return nil, errs.ErrMappingSourceIDRequired
	}

	for _, snap := range c.entries.All() {
		if snap.SourceID == sourceID && snap.Fingerprint == fingerprint {
			return snap, nil
		}
	}

	d, err := c.descriptors.GetByFingerprint(ctx, sourceID, fingerprint)
	if err != nil {
		if errors.Is(err, errs.ErrProtoDescriptorNotFound) {
			return nil, errs.ErrMappingDescriptorMissing
		}
		return nil, coreerrs.Wrap(err, "registry: lookup by fingerprint")
	}
	return c.GetOrBuild(ctx, d.SourceID, d.Revision)
}

// Invalidate drops the cached snapshot for (sourceID, revision) if present.
func (c *Cache) Invalidate(sourceID, revision string) {
	c.entries.Remove(snapshotKey(sourceID, revision))
}

// InvalidateSource drops all cached snapshots belonging to the given source.
func (c *Cache) InvalidateSource(sourceID string) {
	if sourceID == "" {
		return
	}
	prefix := sourceID + "\x00"
	for k := range c.entries.Keys() {
		if strings.HasPrefix(k, prefix) {
			c.entries.Remove(k)
		}
	}
}

func snapshotKey(sourceID, revision string) string {
	return sourceID + "\x00" + revision
}
