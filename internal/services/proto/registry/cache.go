// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dmit-4884/natscope/internal/entities"
	"github.com/dmit-4884/natscope/internal/errs"
	"github.com/dmit-4884/natscope/internal/pkg/protoutils"

	"golang.org/x/sync/singleflight"

	coreerrs "github.com/altessa-s/go-atlas/core/errors"
)

// DescriptorsLookup is the minimal storage contract the cache needs, defined
// locally to keep this package free of storage-layer imports.
type DescriptorsLookup interface {
	GetBySourceTag(ctx context.Context, sourceID, tag string) (*entities.ProtoDescriptor, error)
	FindByFingerprint(ctx context.Context, fingerprint string) (*entities.ProtoDescriptor, error)
}

// maxCachedSnapshots bounds the cache. Snapshots hold parsed descriptor sets,
// and nothing evicts them on its own: a long session that browses many
// source/tag pairs would otherwise grow without limit.
const maxCachedSnapshots = 32

// Cache is a thread-safe per-snapshot registry cache; singleflight guards
// lookups against thundering-herd parse on a cold key.
type Cache struct {
	descriptors DescriptorsLookup

	mu      sync.RWMutex
	entries map[string]*Snapshot

	sf singleflight.Group
}

// NewCache constructs a new Cache wrapping the given descriptor storage.
func NewCache(descriptors DescriptorsLookup) *Cache {
	return &Cache{
		descriptors: descriptors,
		entries:     make(map[string]*Snapshot),
	}
}

// GetOrBuild returns the cached snapshot for (sourceID, tag), parsing from
// storage on miss; concurrent calls share one parse via singleflight.
func (c *Cache) GetOrBuild(ctx context.Context, sourceID, tag string) (*Snapshot, error) {
	if sourceID == "" {
		return nil, errs.ErrMappingSourceNotFound
	}
	if tag == "" {
		return nil, errs.ErrMappingDescriptorMissing
	}

	key := snapshotKey(sourceID, tag)

	c.mu.RLock()
	snap := c.entries[key]
	c.mu.RUnlock()
	if snap != nil {
		return snap, nil
	}

	v, err, _ := c.sf.Do(key, func() (any, error) {
		// Re-check: another caller may have populated while we waited.
		c.mu.RLock()
		cached := c.entries[key]
		c.mu.RUnlock()
		if cached != nil {
			return cached, nil
		}

		d, err := c.descriptors.GetBySourceTag(ctx, sourceID, tag)
		if err != nil {
			if errors.Is(err, errs.ErrProtoDescriptorNotFound) {
				return nil, errs.ErrMappingDescriptorMissing
			}
			return nil, coreerrs.Wrap(err, "registry: load descriptor")
		}
		if len(d.DescriptorSet) == 0 {
			return nil, errs.ErrMappingDescriptorMissing
		}

		messages, err := protoutils.ParseDescriptorSet(d.DescriptorSet)
		if err != nil {
			return nil, coreerrs.Wrap(err, "registry: parse descriptor")
		}

		built := &Snapshot{
			SourceID:    sourceID,
			Tag:         tag,
			Descriptor:  d,
			Messages:    messages,
			ParsedAt:    time.Now(),
			Fingerprint: fingerprint(d.DescriptorSet),
		}

		c.mu.Lock()
		c.evictLocked()
		c.entries[key] = built
		c.mu.Unlock()

		return built, nil
	})
	if err != nil {
		return nil, err
	}
	snap, ok := v.(*Snapshot)
	if !ok {
		return nil, fmt.Errorf("%w: %T", errs.ErrUnexpectedSingleflightType, v)
	}
	return snap, nil
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

	c.mu.RLock()
	for _, snap := range c.entries {
		if snap.SourceID == sourceID && snap.Fingerprint == fingerprint {
			c.mu.RUnlock()
			return snap, nil
		}
	}
	c.mu.RUnlock()

	d, err := c.descriptors.FindByFingerprint(ctx, fingerprint)
	if err != nil {
		if errors.Is(err, errs.ErrProtoDescriptorNotFound) {
			return nil, errs.ErrMappingDescriptorMissing
		}
		return nil, coreerrs.Wrap(err, "registry: lookup by fingerprint")
	}
	if d.SourceID != sourceID {
		// Fingerprint matched a different source; refuse — never substitute across
		// sources even on coinciding content hash.
		return nil, errs.ErrMappingDescriptorMissing
	}
	return c.GetOrBuild(ctx, d.SourceID, d.Tag)
}

// Invalidate drops the cached snapshot for (sourceID, tag) if present.
func (c *Cache) Invalidate(sourceID, tag string) {
	key := snapshotKey(sourceID, tag)
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

// InvalidateSource drops all cached snapshots belonging to the given source.
func (c *Cache) InvalidateSource(sourceID string) {
	if sourceID == "" {
		return
	}
	prefix := sourceID + "\x00"
	c.mu.Lock()
	for k := range c.entries {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.entries, k)
		}
	}
	c.mu.Unlock()
}

// evictLocked drops the oldest snapshots until there is room for one more.
// Caller holds c.mu.
func (c *Cache) evictLocked() {
	for len(c.entries) >= maxCachedSnapshots {
		oldestKey := ""
		var oldestAt time.Time
		for k, snap := range c.entries {
			if oldestKey == "" || snap.ParsedAt.Before(oldestAt) {
				oldestKey, oldestAt = k, snap.ParsedAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(c.entries, oldestKey)
	}
}

// snapshotKey produces the cache key for (sourceID, tag); NUL separator keeps
// it unambiguous if a tag contains ':'.
func snapshotKey(sourceID, tag string) string {
	return sourceID + "\x00" + tag
}
