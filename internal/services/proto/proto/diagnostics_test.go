// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package proto

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dmit-4884/natscope/internal/pkg/protoutils"
)

func TestAutoWalkHint(t *testing.T) {
	layout := &protoutils.ResolvedLayout{
		Origin: protoutils.RootsOriginInferred,
		Roots:  []string{"proto"},
		Srcs: map[string]string{
			"common/types.proto": "x",
			"api/svc.proto":      "y",
		},
	}

	t.Run("mixed import style detected by basename", func(t *testing.T) {
		h := autoWalkHint("proto/common/types.proto", layout)
		assert.Contains(t, h, `"common/types.proto"`)
		assert.Contains(t, h, "proto") // mentions a detected root
	})

	t.Run("wkt missing", func(t *testing.T) {
		h := autoWalkHint("google/protobuf/timestamp.proto", layout)
		assert.Contains(t, h, "well-known")
	})

	t.Run("genuinely missing", func(t *testing.T) {
		h := autoWalkHint("nowhere/else.proto", layout)
		assert.Contains(t, h, "ImportRoots")
	})
}
