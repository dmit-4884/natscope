// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"go.etcd.io/bbolt"

	"github.com/altessa-s/go-atlas/core/errors"
)

// Store is a JSON-document store over one bbolt bucket. It gives a storage
// shared CRUD, keyset List, and equality lookups over model M.
type Store[M any, PM Doc[M]] struct {
	db       *DB
	bucket   []byte
	notFound error
	sortBy   string
	uniques  []Index
}

// Open creates the bucket if missing (idempotent) and returns the store.
func Open[M any, PM Doc[M]](ctx context.Context, db *DB, spec Spec) (*Store[M, PM], error) {
	sortBy := spec.SortBy
	if sortBy == "" {
		sortBy = SortCreated
	}
	if sortBy != SortCreated && sortBy != SortUpdated {
		return nil, fmt.Errorf("bbstore: invalid SortBy %q", sortBy)
	}

	s := &Store[M, PM]{
		db:       db,
		bucket:   []byte(spec.Bucket),
		notFound: spec.NotFound,
		sortBy:   sortBy,
	}
	for _, idx := range spec.Indexes {
		if idx.Unique {
			s.uniques = append(s.uniques, idx)
		}
	}

	if err := db.update(ctx, func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(s.bucket)
		return err
	}); err != nil {
		return nil, errors.WrapOperation(err, "create bucket "+spec.Bucket)
	}
	return s, nil
}

// Save writes a new document, enforcing any Unique index in the same
// transaction. A collision returns [ErrUniqueViolation].
func (s *Store[M, PM]) Save(ctx context.Context, m PM) error {
	raw, err := s.marshal(m)
	if err != nil {
		return err
	}
	return s.db.update(ctx, func(tx *bbolt.Tx) error {
		if err := s.checkUnique(tx, m, raw, ""); err != nil {
			return err
		}
		return tx.Bucket(s.bucket).Put([]byte(m.DocBase().ID), raw)
	})
}

// Upsert writes the document, replacing any existing one with the same id. Used
// by singleton stores; it does not run uniqueness checks.
func (s *Store[M, PM]) Upsert(ctx context.Context, m PM) error {
	raw, err := s.marshal(m)
	if err != nil {
		return err
	}
	return s.db.update(ctx, func(tx *bbolt.Tx) error {
		return tx.Bucket(s.bucket).Put([]byte(m.DocBase().ID), raw)
	})
}

// Update overwrites an active document, enforcing Unique indexes (excluding
// itself); returns the not-found sentinel if no active document matches id.
func (s *Store[M, PM]) Update(ctx context.Context, m PM) error {
	raw, err := s.marshal(m)
	if err != nil {
		return err
	}
	id := m.DocBase().ID
	return s.db.update(ctx, func(tx *bbolt.Tx) error {
		existing := tx.Bucket(s.bucket).Get([]byte(id))
		if existing == nil {
			return s.notFound
		}
		em, err := s.decode(existing)
		if err != nil {
			return err
		}
		if em.DocBase().DeletedAt != nil {
			return s.notFound
		}
		if err := s.checkUnique(tx, m, raw, id); err != nil {
			return err
		}
		return tx.Bucket(s.bucket).Put([]byte(id), raw)
	})
}

// SoftDelete stamps deletedAt/updatedAt inside the document so a later read with
// includeDeleted still sees the deletion. The timestamp comes from the caller.
func (s *Store[M, PM]) SoftDelete(ctx context.Context, id string, newUpdatedAt int64) error {
	return s.db.update(ctx, func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		raw := b.Get([]byte(id))
		if raw == nil {
			return s.notFound
		}
		m, err := s.decode(raw)
		if err != nil {
			return err
		}
		base := m.DocBase()
		if base.DeletedAt != nil {
			return s.notFound
		}
		base.DeletedAt = &newUpdatedAt
		base.UpdatedAt = newUpdatedAt
		out, err := s.marshal(m)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), out)
	})
}

// Delete removes a document outright. It returns the not-found sentinel if
// nothing matched.
func (s *Store[M, PM]) Delete(ctx context.Context, id string) error {
	return s.db.update(ctx, func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		if b.Get([]byte(id)) == nil {
			return s.notFound
		}
		return b.Delete([]byte(id))
	})
}

// DeleteAll removes every document and returns how many were removed.
func (s *Store[M, PM]) DeleteAll(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.update(ctx, func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.bucket)
		keys := collectKeys(b)
		n = int64(len(keys))
		for _, k := range keys {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, errors.WrapOperation(err, "delete all "+string(s.bucket))
	}
	return n, nil
}

// Get returns a document by id. A soft-deleted document is hidden unless
// includeDeleted is true.
func (s *Store[M, PM]) Get(ctx context.Context, id string, includeDeleted ...bool) (PM, error) {
	var out PM
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		raw := tx.Bucket(s.bucket).Get([]byte(id))
		if raw == nil {
			return s.notFound
		}
		m, err := s.decode(raw)
		if err != nil {
			return err
		}
		if m.DocBase().DeletedAt != nil && !want(includeDeleted) {
			return s.notFound
		}
		out = m
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetBy returns the first document whose field at path equals val.
func (s *Store[M, PM]) GetBy(ctx context.Context, path string, val any, includeDeleted ...bool) (PM, error) {
	var out PM
	found := false
	matcher := newFieldMatcher(path, val)
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		return s.scan(tx, func(raw []byte, m PM) (bool, error) {
			if m.DocBase().DeletedAt != nil && !want(includeDeleted) {
				return true, nil
			}
			if matcher.matches(raw) {
				out, found = m, true
				return false, nil
			}
			return true, nil
		})
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, s.notFound
	}
	return out, nil
}

// Exists reports whether a document with the id exists. A soft-deleted document
// counts only when includeDeleted is true.
func (s *Store[M, PM]) Exists(ctx context.Context, id string, includeDeleted ...bool) (bool, error) {
	ok := false
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		raw := tx.Bucket(s.bucket).Get([]byte(id))
		if raw == nil {
			return nil
		}
		if !want(includeDeleted) {
			m, err := s.decode(raw)
			if err != nil {
				return err
			}
			if m.DocBase().DeletedAt != nil {
				return nil
			}
		}
		ok = true
		return nil
	})
	return ok, err
}

// CountActiveBy counts active documents whose field at path equals val,
// excluding id excludeID (pass "" to count all); used for uniqueness checks.
func (s *Store[M, PM]) CountActiveBy(ctx context.Context, path string, val any, excludeID string) (int64, error) {
	var n int64
	matcher := newFieldMatcher(path, val)
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		return s.scan(tx, func(raw []byte, m PM) (bool, error) {
			base := m.DocBase()
			if base.DeletedAt != nil || base.ID == excludeID {
				return true, nil
			}
			if matcher.matches(raw) {
				n++
			}
			return true, nil
		})
	})
	return n, err
}

// List returns one keyset page (newest first by the store's sort field) plus the
// next cursor and, when asked, the active total.
func (s *Store[M, PM]) List(ctx context.Context, cursor string, limit int64, includeTotal bool) ([]PM, string, *int64, error) {
	return s.ListFiltered(ctx, nil, cursor, limit, includeTotal)
}

// ListFiltered is List with extra equality filters, all ANDed. A nil/empty slice
// lists everything active.
func (s *Store[M, PM]) ListFiltered(
	ctx context.Context,
	filters []Filter,
	cursorStr string,
	limit int64,
	includeTotal bool,
) ([]PM, string, *int64, error) {
	cur, err := parseCursor(cursorStr)
	if err != nil {
		return nil, "", nil, err
	}

	matchers := make([]fieldMatcher, len(filters))
	for i, f := range filters {
		matchers[i] = newFieldMatcher(f.Path, f.Val)
	}

	var all []PM
	err = s.db.view(ctx, func(tx *bbolt.Tx) error {
		return s.scan(tx, func(raw []byte, m PM) (bool, error) {
			if m.DocBase().DeletedAt != nil {
				return true, nil
			}
			for i := range matchers {
				if !matchers[i].matches(raw) {
					return true, nil
				}
			}
			all = append(all, m)
			return true, nil
		})
	})
	if err != nil {
		return nil, "", nil, errors.WrapOperation(err, "list "+string(s.bucket))
	}

	sort.Slice(all, func(i, j int) bool {
		si, sj := s.sortValue(all[i]), s.sortValue(all[j])
		if si != sj {
			return si > sj
		}
		return all[i].DocBase().ID > all[j].DocBase().ID
	})

	total := int64(len(all))

	if cur.has == 1 {
		var rest []PM
		for _, m := range all {
			sv, id := s.sortValue(m), m.DocBase().ID
			if sv < cur.sort || (sv == cur.sort && id < cur.id) {
				rest = append(rest, m)
			}
		}
		all = rest
	}

	var next string
	if limit > 0 && int64(len(all)) > limit {
		last := all[limit-1]
		next = encodeCursor(s.sortValue(last), last.DocBase().ID)
		all = all[:limit]
	}

	var totalPtr *int64
	if includeTotal {
		totalPtr = &total
	}
	return all, next, totalPtr, nil
}

// ListBy returns all active documents whose field at path equals val, oldest
// first.
func (s *Store[M, PM]) ListBy(ctx context.Context, path string, val any) ([]PM, error) {
	var out []PM
	matcher := newFieldMatcher(path, val)
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		return s.scan(tx, func(raw []byte, m PM) (bool, error) {
			if m.DocBase().DeletedAt == nil && matcher.matches(raw) {
				out = append(out, m)
			}
			return true, nil
		})
	})
	if err != nil {
		return nil, err
	}
	sortByCreatedAsc(out)
	return out, nil
}

// ListAll returns every active document, oldest first.
func (s *Store[M, PM]) ListAll(ctx context.Context) ([]PM, error) {
	var out []PM
	err := s.db.view(ctx, func(tx *bbolt.Tx) error {
		return s.scan(tx, func(_ []byte, m PM) (bool, error) {
			if m.DocBase().DeletedAt == nil {
				out = append(out, m)
			}
			return true, nil
		})
	})
	if err != nil {
		return nil, err
	}
	sortByCreatedAsc(out)
	return out, nil
}

// DeleteBy removes all documents whose field at path equals val (including
// soft-deleted ones) and returns the count removed.
func (s *Store[M, PM]) DeleteBy(ctx context.Context, path string, val any) (int64, error) {
	var n int64
	matcher := newFieldMatcher(path, val)
	err := s.db.update(ctx, func(tx *bbolt.Tx) error {
		var ids [][]byte
		if err := s.scan(tx, func(raw []byte, m PM) (bool, error) {
			if matcher.matches(raw) {
				ids = append(ids, []byte(m.DocBase().ID))
			}
			return true, nil
		}); err != nil {
			return err
		}
		n = int64(len(ids))
		b := tx.Bucket(s.bucket)
		for _, id := range ids {
			if err := b.Delete(id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, errors.WrapOperation(err, "delete by "+string(s.bucket))
	}
	return n, nil
}

// WithTransaction runs fn in one transaction; a transaction already on ctx is
// reused.
func (s *Store[M, PM]) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return s.db.WithTransaction(ctx, fn)
}

// scan walks every document in the bucket, decoding each and handing the raw
// JSON and decoded model to cb. cb returns false to stop early.
func (s *Store[M, PM]) scan(tx *bbolt.Tx, cb func(raw []byte, m PM) (bool, error)) error {
	b := tx.Bucket(s.bucket)
	if b == nil {
		return nil
	}
	c := b.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		m, err := s.decode(v)
		if err != nil {
			return err
		}
		cont, err := cb(v, m)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}
	return nil
}

// checkUnique rejects a write whose value collides with another document under
// any Unique index. A soft-deleted candidate holds no value, so it is skipped.
//
// This scans the whole bucket per unique index on every write. That is fine for
// this app's local, low-cardinality domains (hundreds of documents); do not
// reuse bbstore for high-cardinality data without adding a secondary index.
func (s *Store[M, PM]) checkUnique(tx *bbolt.Tx, m PM, raw []byte, excludeID string) error {
	if len(s.uniques) == 0 || m.DocBase().DeletedAt != nil {
		return nil
	}
	for _, idx := range s.uniques {
		want, ok := extractField(raw, idx.Path)
		if !ok {
			continue
		}
		conflict := false
		err := s.scan(tx, func(other []byte, om PM) (bool, error) {
			if om.DocBase().ID == excludeID {
				return true, nil
			}
			if idx.ActiveOnly && om.DocBase().DeletedAt != nil {
				return true, nil
			}
			if of, ok := extractField(other, idx.Path); ok && jsonEqual(of, want) {
				conflict = true
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return err
		}
		if conflict {
			return &ErrUniqueViolation{Bucket: string(s.bucket), Path: idx.Path, Value: tokenString(want)}
		}
	}
	return nil
}

func (s *Store[M, PM]) marshal(m PM) ([]byte, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, errors.WrapOperation(err, "marshal "+string(s.bucket))
	}
	return raw, nil
}

func (s *Store[M, PM]) decode(raw []byte) (PM, error) {
	m := PM(new(M))
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, errors.WrapOperation(err, "unmarshal "+string(s.bucket))
	}
	return m, nil
}

func (s *Store[M, PM]) sortValue(m PM) int64 {
	if s.sortBy == SortUpdated {
		return m.DocBase().UpdatedAt
	}
	return m.DocBase().CreatedAt
}

func sortByCreatedAsc[M any, PM Doc[M]](items []PM) {
	sort.Slice(items, func(i, j int) bool {
		bi, bj := items[i].DocBase(), items[j].DocBase()
		if bi.CreatedAt != bj.CreatedAt {
			return bi.CreatedAt < bj.CreatedAt
		}
		return bi.ID < bj.ID
	})
}

// collectKeys copies every key in the bucket. bbolt forbids deleting through a
// live cursor, so callers delete from the returned slice.
func collectKeys(b *bbolt.Bucket) [][]byte {
	var keys [][]byte
	c := b.Cursor()
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		keys = append(keys, append([]byte(nil), k...))
	}
	return keys
}

// splitPath turns a "$.a.b" or "a.b" path into its field segments.
func splitPath(path string) []string {
	p := strings.TrimPrefix(path, "$")
	p = strings.TrimPrefix(p, ".")
	return strings.Split(p, ".")
}

// extractField returns the raw JSON token at path within a document, navigating
// nested objects. The second result is false when any segment is absent.
func extractField(raw []byte, path string) (json.RawMessage, bool) {
	cur := json.RawMessage(raw)
	for _, key := range splitPath(path) {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(cur, &obj); err != nil {
			return nil, false
		}
		v, ok := obj[key]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// fieldMatcher tests documents against a field==value predicate; the compare
// value is marshaled once at construction and reused across the whole scan.
type fieldMatcher struct {
	path string
	want []byte
	ok   bool // false when the value failed to marshal → matches nothing
}

// newFieldMatcher marshals val once for reuse across a scan.
func newFieldMatcher(path string, val any) fieldMatcher {
	wb, err := json.Marshal(val)
	return fieldMatcher{path: path, want: wb, ok: err == nil}
}

// matches reports whether the document's field at path equals the bound value.
func (m fieldMatcher) matches(raw []byte) bool {
	if !m.ok {
		return false
	}
	field, ok := extractField(raw, m.path)
	return ok && jsonEqual(field, m.want)
}

// jsonEqual compares two JSON tokens by decoded value, so 7 and 7.0 or differing
// whitespace still match.
func jsonEqual(a, b []byte) bool {
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(x, y)
}

// tokenString renders a JSON token for an error message, unquoting plain strings.
func tokenString(tok []byte) string {
	var s string
	if json.Unmarshal(tok, &s) == nil {
		return s
	}
	return string(tok)
}

// want reports whether the optional includeDeleted flag is set.
func want(includeDeleted []bool) bool {
	return len(includeDeleted) > 0 && includeDeleted[0]
}
