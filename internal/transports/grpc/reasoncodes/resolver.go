// Copyright 2026 The Natscope Authors
// SPDX-License-Identifier: Apache-2.0

package reasoncodes

import (
	"strings"

	corestrings "github.com/altessa-s/go-atlas/core/text/strings"
)

// Canonical reason codes for standard protovalidate rules. go-atlas no longer
// maps these, so the service owns them.
const (
	InvalidMinLengthOrValue = "INVALID_MIN_LENGTH_OR_VALUE"
	InvalidMaxLengthOrValue = "INVALID_MAX_LENGTH_OR_VALUE"
	InvalidFormatUUID       = "INVALID_FORMAT_UUID"
	InvalidFormatEmail      = "INVALID_FORMAT_EMAIL"
	InvalidFormatRegex      = "INVALID_FORMAT_REGEX"
	InvalidFormatURL        = "INVALID_FORMAT_URL"
	InvalidEnumValue        = "INVALID_ENUM_VALUE"
)

const (
	// ruleIDRequired is protovalidate's rule ID for a required-field violation.
	// Its code is derived from the field name (FIELD_NAME_REQUIRED), not a fixed
	// constant.
	ruleIDRequired = "required"
	// requiredSuffix is appended to the screaming-snake field name.
	requiredSuffix = "_REQUIRED"
	// requiredFallback is used when the field name cannot be extracted.
	requiredFallback = "REQUIRED"
)

// standardFullID maps full rule IDs whose trailing segment is ambiguous.
var standardFullID = map[string]string{
	"enum.defined_only": InvalidEnumValue,
	"enum.in":           InvalidEnumValue,
	"enum.const":        InvalidEnumValue,
	"string.in":         InvalidEnumValue,
	"string.uuid":       InvalidFormatUUID,
	"string.email":      InvalidFormatEmail,
	"string.pattern":    InvalidFormatRegex,
	"bytes.pattern":     InvalidFormatRegex,
	"string.uri":        InvalidFormatURL,
	"string.uri_ref":    InvalidFormatURL,
}

// standardSuffix maps numeric/size families by the rule ID's last segment.
// Combined range rules (gte_lte, ...) are omitted: the seam has no field value
// to tell min from max, so a field needing both bounds must split them in proto.
var standardSuffix = map[string]string{
	"gte":       InvalidMinLengthOrValue,
	"gt":        InvalidMinLengthOrValue,
	"min_len":   InvalidMinLengthOrValue,
	"min_items": InvalidMinLengthOrValue,
	"min_pairs": InvalidMinLengthOrValue,
	"min_bytes": InvalidMinLengthOrValue,
	"lte":       InvalidMaxLengthOrValue,
	"lt":        InvalidMaxLengthOrValue,
	"max_len":   InvalidMaxLengthOrValue,
	"max_items": InvalidMaxLengthOrValue,
	"max_pairs": InvalidMaxLengthOrValue,
	"max_bytes": InvalidMaxLengthOrValue,
	"len":       InvalidMaxLengthOrValue,
	"len_bytes": InvalidMaxLengthOrValue,
}

// Resolver maps a protovalidate rule ID to a canonical reason code, checking the
// domain catalog before the standard rules. It satisfies
// bufhelpers.ReasonCodeResolver.
type Resolver struct {
	catalog  map[string]string
	prefixes []string
}

// NewResolver builds a Resolver over catalog and the service-owned rule prefixes.
func NewResolver(catalog map[string]string, prefixes ...string) *Resolver {
	return &Resolver{catalog: catalog, prefixes: prefixes}
}

// Resolve returns the reason code for ruleID, or "" when none applies (never
// "UNKNOWN"). A required violation derives FIELD_NAME_REQUIRED from fieldName; an
// owned-prefix rule absent from the catalog resolves to "".
func (r *Resolver) Resolve(ruleID, fieldName string) string {
	if ruleID == "" {
		return ""
	}
	if code, ok := r.catalog[ruleID]; ok {
		return code
	}
	if ruleID == ruleIDRequired {
		if fieldName == "" {
			return requiredFallback
		}
		return corestrings.ToScreamingSnakeCase(fieldName) + requiredSuffix
	}
	for _, p := range r.prefixes {
		if strings.HasPrefix(ruleID, p) {
			return ""
		}
	}
	if code, ok := standardFullID[ruleID]; ok {
		return code
	}
	if code, ok := standardSuffix[ruleSuffix(ruleID)]; ok {
		return code
	}
	return ""
}

// ruleSuffix returns the segment of ruleID after the last ".".
func ruleSuffix(ruleID string) string {
	if i := strings.LastIndexByte(ruleID, '.'); i >= 0 {
		return ruleID[i+1:]
	}
	return ruleID
}
