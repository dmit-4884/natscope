// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	berrors "go.etcd.io/bbolt/errors"
)

// openTimeout bounds how long NewDB waits for the file lock before failing.
const openTimeout = 5 * time.Second

// fileMode is the permission for a freshly created database file.
const fileMode = 0o600

// DB wraps a bbolt database. One file holds every store as a top-level bucket.
// Writes serialize through one writer; reads use concurrent MVCC snapshots.
type DB struct {
	b *bbolt.DB
}

// NewDB opens the bbolt file at path, creating it if needed.
func NewDB(path string) (*DB, error) {
	b, err := bbolt.Open(path, fileMode, &bbolt.Options{Timeout: openTimeout})
	if err != nil {
		return nil, describeOpenError(err, path)
	}
	return &DB{b: b}, nil
}

// describeOpenError turns bbolt's terse open failures into actionable
// messages: the two realistic field cases are a second running instance
// (file lock) and a damaged file.
func describeOpenError(err error, path string) error {
	switch {
	case errors.Is(err, berrors.ErrTimeout):
		return fmt.Errorf(
			"database file %s is locked — is another natscope instance already running? "+
				"Stop it (or remove a stale lock left by a killed process) and try again: %w",
			path, err)
	case errors.Is(err, berrors.ErrInvalid),
		errors.Is(err, berrors.ErrVersionMismatch),
		errors.Is(err, berrors.ErrChecksum):
		return fmt.Errorf(
			"database file %s appears to be corrupted: %w. "+
				"Move the file aside to start fresh — saved connections, proto sources "+
				"and settings will be lost (secrets stay in the OS keychain)",
			path, err)
	default:
		return fmt.Errorf("open bbolt database %s: %w", path, err)
	}
}

// Close flushes and releases the file lock.
func (db *DB) Close() error {
	if db.b == nil {
		return nil
	}
	return db.b.Close()
}

// txKeyType is the context key for an in-flight write transaction.
type txKeyType struct{}

var txKey txKeyType

// WithTransaction runs fn inside one read-write transaction. A transaction
// already on ctx is reused, so nested calls join the outer commit/rollback.
func (db *DB) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if tx, ok := ctx.Value(txKey).(*bbolt.Tx); ok && tx != nil {
		return fn(ctx)
	}
	return db.b.Update(func(tx *bbolt.Tx) error {
		return fn(context.WithValue(ctx, txKey, tx))
	})
}

// view runs fn against the active transaction on ctx, or a fresh read-only one.
func (db *DB) view(ctx context.Context, fn func(tx *bbolt.Tx) error) error {
	if tx, ok := ctx.Value(txKey).(*bbolt.Tx); ok && tx != nil {
		return fn(tx)
	}
	return db.b.View(fn)
}

// update runs fn against the active transaction on ctx, or a fresh read-write
// one.
func (db *DB) update(ctx context.Context, fn func(tx *bbolt.Tx) error) error {
	if tx, ok := ctx.Value(txKey).(*bbolt.Tx); ok && tx != nil {
		return fn(tx)
	}
	return db.b.Update(fn)
}

// GetRaw returns a copy of the bytes stored for key in bucket, or nil if
// either is absent; for migration tooling/diagnostics, stores never use it.
func (db *DB) GetRaw(ctx context.Context, bucket, key string) ([]byte, error) {
	var out []byte
	err := db.view(ctx, func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}
		if v := b.Get([]byte(key)); v != nil {
			out = append([]byte(nil), v...)
		}
		return nil
	})
	return out, err
}
