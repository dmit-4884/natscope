// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import "fmt"

// ErrUniqueViolation reports a write rejected by a Unique index. A domain maps it
// to its own sentinel, for example errs.ErrConnectionNameAlreadyInUse.
type ErrUniqueViolation struct {
	Bucket string
	Path   string
	Value  string
}

func (e *ErrUniqueViolation) Error() string {
	return fmt.Sprintf("bbstore: unique violation on %s.%s = %q", e.Bucket, e.Path, e.Value)
}
