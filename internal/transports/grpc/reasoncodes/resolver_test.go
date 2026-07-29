// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package reasoncodes

import (
	"testing"

	"github.com/stretchr/testify/assert"

	bufhelpers "github.com/altessa-s/go-atlas/transport/grpc/interceptors/protovalidator/buf"
)

// Resolver must satisfy the seam the validator injects it through.
var _ bufhelpers.ReasonCodeResolver = (*Resolver)(nil)

func TestResolver_Resolve(t *testing.T) {
	t.Parallel()
	r := NewResolver(Catalog, Prefix)

	tests := []struct {
		name      string
		ruleID    string
		fieldName string
		want      string
	}{
		{"empty rule id", "", "connection_id", ""},

		// Domain catalog wins over everything else.
		{"catalog cross-field", "natscope.nats.messages.start_seq_xor_start_time", "", "MUTUALLY_EXCLUSIVE_FIELDS"},

		// Required derives the code from the field name.
		{"required with field", ruleIDRequired, "connection_id", "CONNECTION_ID_REQUIRED"},
		{"required nested leaf", ruleIDRequired, "sourceId", "SOURCE_ID_REQUIRED"},
		{"required without field", ruleIDRequired, "", requiredFallback},

		// Owned-prefix rules absent from the catalog resolve to "" — the guard
		// beats standard-suffix inference (natscope.foo.gte does NOT become a MIN).
		{"owned prefix uncataloged", "natscope.foo.gte", "", ""},

		// Standard min/max families by suffix.
		{"string.min_len", "string.min_len", "stream_name", InvalidMinLengthOrValue},
		{"int32.gte", "int32.gte", "limit", InvalidMinLengthOrValue},
		{"uint64.gt", "uint64.gt", "seq", InvalidMinLengthOrValue},
		{"repeated.min_items", "repeated.min_items", "items", InvalidMinLengthOrValue},
		{"bytes.min_len", "bytes.min_len", "payload", InvalidMinLengthOrValue},
		{"int32.lte", "int32.lte", "rate", InvalidMaxLengthOrValue},
		{"string.max_len", "string.max_len", "name", InvalidMaxLengthOrValue},

		// Standard rules by full id (ambiguous suffix).
		{"string.uuid", "string.uuid", "id", InvalidFormatUUID},
		{"string.in", "string.in", "format", InvalidEnumValue},
		{"enum.defined_only", "enum.defined_only", "direction", InvalidEnumValue},

		// Unknown / unmapped rules yield no code (never "UNKNOWN").
		{"combined range unmapped", "int32.gte_lte", "year", ""},
		{"unknown suffix", "string.mystery", "x", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, r.Resolve(tt.ruleID, tt.fieldName))
		})
	}
}
