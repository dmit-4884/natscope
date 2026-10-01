// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/altessa-s/go-atlas/domain/converter"

	"github.com/dmit-4884/natscope/internal/entities"
)

func TestViews(t *testing.T) {
	t.Parallel()

	conflict := converter.Convert(&entities.SchemaConflict{
		Kind: entities.ConflictSameShape, Severity: entities.SeverityInfo, Symbol: "a.B",
		First: entities.SchemaRef{SourceID: "s1", File: "a.proto"}, Second: entities.SchemaRef{SourceID: "s2", Revision: "v2", File: "a.proto"},
	}, &conflictView{})
	assert.Equal(t, "same_shape", conflict.Kind)
	assert.Equal(t, "info", conflict.Severity)
	assert.Equal(t, "s2", conflict.Second.SourceID)
	assert.Equal(t, "v2", conflict.Second.Revision)

	res := converter.Convert(&entities.ValidationResult{Violations: []*entities.ValidationViolation{
		{FieldPath: "id", Message: "value is required", ConstraintId: "required"},
	}}, &validateOutput{})
	assert.False(t, res.Valid)
	assert.Equal(t, []violationView{{FieldPath: "id", Message: "value is required", ConstraintId: "required"}}, res.Violations)

	pinned := "v1.2.0"
	mapping := converter.Convert(&entities.SubjectMapping{Pattern: "orders.*", MessageType: "o.v1.Order", SourceID: "s1", PinnedFingerprint: &pinned},
		&mappingView{})
	assert.Equal(t, &mappingView{Pattern: "orders.*", MessageType: "o.v1.Order", SourceID: "s1", PinnedFingerprint: &pinned}, mapping)
}
