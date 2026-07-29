// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package entities

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// BaseEntity holds identity, timestamps, soft-delete, opaque ETag for OCC.
type BaseEntity struct {
	Id string

	// Etag bumped on every mutation; Update/Delete rejected if stored Etag has
	// moved (OCC).
	Etag string

	CreatedAt time.Time

	UpdatedAt time.Time

	// DeletedAt is the soft-delete time, nil when active.
	DeletedAt *time.Time
}

// New returns a BaseEntity with a unique Id, fresh ETag, and current
// timestamps.
func New() *BaseEntity {
	now := time.Now().UTC()
	return &BaseEntity{
		Id:        uuid.New().String(),
		Etag:      generateEtag(),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsDeleted checks if the entity is soft deleted.
func (e *BaseEntity) IsDeleted() bool {
	return e.DeletedAt != nil
}

// UpdateTimestamps refreshes UpdatedAt; CreatedAt set only if zero (preserves
// original).
func (e *BaseEntity) UpdateTimestamps() {
	now := time.Now().UTC()
	e.UpdatedAt = now

	if e.CreatedAt.IsZero() {
		e.CreatedAt = now
	}
}

// UpdateEtag regenerates the ETag; invoke on any OCC-participating state
// transition.
func (e *BaseEntity) UpdateEtag() {
	e.Etag = generateEtag()
}

// BeforeUpdate refreshes timestamps and ETag; call before handing the entity to
// storage.
func (e *BaseEntity) BeforeUpdate() {
	e.UpdateTimestamps()
	e.UpdateEtag()
}

// Restore clears the soft-delete flag and refreshes timestamps + ETag; identity
// unchanged.
func (e *BaseEntity) Restore() {
	e.DeletedAt = nil
	e.UpdateTimestamps()
	e.UpdateEtag()
}

// GetID returns the entity's Id.
func (e *BaseEntity) GetID() string {
	return e.Id
}

// Equal reports field-by-field equality (tests and change detection).
func (e *BaseEntity) Equal(other *BaseEntity) bool {
	if e == nil || other == nil {
		return e == other
	}
	if e.Id != other.Id || e.Etag != other.Etag {
		return false
	}
	if !e.CreatedAt.Equal(other.CreatedAt) || !e.UpdatedAt.Equal(other.UpdatedAt) {
		return false
	}
	switch {
	case e.DeletedAt == nil && other.DeletedAt == nil:
		return true
	case e.DeletedAt == nil || other.DeletedAt == nil:
		return false
	default:
		return e.DeletedAt.Equal(*other.DeletedAt)
	}
}

// generateEtag returns a fresh opaque ETag: UUIDv4 without hyphens (32 hex
// chars).
func generateEtag() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
