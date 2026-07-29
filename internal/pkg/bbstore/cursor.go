// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package bbstore

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidCursor marks an unparseable pagination cursor. Storages surface it
// unchanged; the transport maps it to a clean invalid-argument status instead
// of leaking the raw parse text to the client.
var ErrInvalidCursor = errors.New("invalid pagination cursor")

// cursorSep separates the sort key and id inside the opaque pagination cursor.
const cursorSep = "\x1f"

// cursorParts is the number of components in a decoded cursor (sort key, id).
const cursorParts = 2

// cursor holds the keyset args decoded from an opaque cursor. The first page
// passes has = 0. The wire format matches the old docstore cursor.
type cursor struct {
	has  int64
	sort int64
	id   string
}

// parseCursor decodes an opaque cursor into keyset args; empty means first page.
func parseCursor(s string) (cursor, error) {
	if s == "" {
		return cursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return cursor{}, fmt.Errorf("decode cursor: %w", ErrInvalidCursor)
	}
	parts := strings.SplitN(string(raw), cursorSep, cursorParts)
	if len(parts) != cursorParts {
		return cursor{}, fmt.Errorf("malformed cursor: %w", ErrInvalidCursor)
	}
	sort, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return cursor{}, fmt.Errorf("cursor sort key: %w", ErrInvalidCursor)
	}
	return cursor{has: 1, sort: sort, id: parts[1]}, nil
}

// encodeCursor builds the opaque cursor from the last item's (sortKey, id).
func encodeCursor(sort int64, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(sort, 10) + cursorSep + id))
}
